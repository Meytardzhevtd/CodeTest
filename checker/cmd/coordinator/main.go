package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v11"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	grpccoordinator "github.com/meytardzhevtd/CodeTest/checker/internal/coordinator"
	"github.com/meytardzhevtd/CodeTest/checker/internal/grpc/judgepb"
	"github.com/meytardzhevtd/CodeTest/pkg/kafka"
	"github.com/meytardzhevtd/CodeTest/pkg/storage"
)

type config struct {
	KafkaBrokers []string `env:"KAFKA_BROKERS" envSeparator:"," envDefault:"localhost:9092"`
	GroupID      string   `env:"KAFKA_GROUP_ID" envDefault:"coordinator"`
	GRPCAddr     string   `env:"GRPC_ADDR" envDefault:":50051"`

	MinioEndpoint  string `env:"MINIO_ENDPOINT,required"`
	MinioAccessKey string `env:"MINIO_ACCESS_KEY,required"`
	MinioSecretKey string `env:"MINIO_SECRET_KEY,required"`
	MinioBucket    string `env:"MINIO_BUCKET,required"`
	MinioUseSSL    bool   `env:"MINIO_USE_SSL"`

	RedisAddr string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := kafka.EnsureTopics(ctx, cfg.KafkaBrokers, kafka.TopicSubmissions, kafka.TopicResults); err != nil {
		log.Fatalf("ensure kafka topics: %v", err)
	}

	producer := kafka.NewProducer(cfg.KafkaBrokers, kafka.TopicResults)
	defer producer.Close()

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, kafka.TopicSubmissions, cfg.GroupID)
	defer consumer.Close()

	testStore, err := storage.NewClient(storage.Config{
		Endpoint:  cfg.MinioEndpoint,
		AccessKey: cfg.MinioAccessKey,
		SecretKey: cfg.MinioSecretKey,
		Bucket:    cfg.MinioBucket,
		UseSSL:    cfg.MinioUseSSL,
	})
	if err != nil {
		log.Fatalf("create minio client: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("[coordinator] redis at %s not reachable yet, test-case cache will retry lazily: %v", cfg.RedisAddr, err)
	}
	testCache := grpccoordinator.NewTestCache(redisClient)

	queue := grpccoordinator.NewQueue(ctx)

	grpcServer := grpc.NewServer()
	judgepb.RegisterCoordinatorServer(grpcServer, grpccoordinator.NewServer(queue, producer, testStore, testCache))

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("listen on %s: %v", cfg.GRPCAddr, err)
	}

	go func() {
		log.Printf("[coordinator] gRPC listening on %s", cfg.GRPCAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	log.Printf("[coordinator] listening on topic %q, group %q", kafka.TopicSubmissions, cfg.GroupID)
	consumer.Consume(ctx, handleSubmission(queue))
}

// handleSubmission разбирает сообщение о сабмишне и кладёт задачу во внутреннюю
// очередь координатора, откуда её заберёт воркер по gRPC. Оффсет коммитится сразу
// после этого (см. Consumer.Consume) — результат работы воркера тут не ждём: если
// координатор упадёт с незавершённой задачей в очереди, она потеряется, и клиенту
// нужно будет отправить решение заново.
func handleSubmission(queue *grpccoordinator.Queue) func(ctx context.Context, data []byte) error {
	return func(ctx context.Context, data []byte) error {
		var msg kafka.SubmissionMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[coordinator] invalid submission message, skipping: %v", err)
			return nil
		}

		if msg.SubmissionID == "" {
			log.Printf("[coordinator] submission message missing submission_id, skipping")
			return nil
		}

		log.Printf("[coordinator] submission %s: получена из kafka (task=%s, language=%s, code=%d bytes)",
			msg.SubmissionID, msg.TaskID, msg.Language, len(msg.Code))

		queue.Enqueue(msg.TaskID, &judgepb.Task{
			SubmissionId: msg.SubmissionID,
			Language:     msg.Language,
			Code:         msg.Code,
		})

		return nil
	}
}
