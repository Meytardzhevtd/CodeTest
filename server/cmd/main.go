package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/meytardzhevtd/CodeTest/pkg/kafka"
	"github.com/meytardzhevtd/CodeTest/pkg/storage"
	authinternal "github.com/meytardzhevtd/CodeTest/server/internal/auth"
	submitinternal "github.com/meytardzhevtd/CodeTest/server/internal/submit"
	tasksinternal "github.com/meytardzhevtd/CodeTest/server/internal/tasks"
)

type config struct {
	DBDSN            string `env:"DB_DSN,required"`
	JWTAccessSecret  string `env:"JWT_ACCESS_SECRET,required"`
	JWTRefreshSecret string `env:"JWT_REFRESH_SECRET,required"`
	JWTIssuer        string `env:"JWT_ISSUER"`

	MinioEndpoint  string `env:"MINIO_ENDPOINT,required"`
	MinioAccessKey string `env:"MINIO_ACCESS_KEY,required"`
	MinioSecretKey string `env:"MINIO_SECRET_KEY,required"`
	MinioBucket    string `env:"MINIO_BUCKET,required"`
	MinioUseSSL    bool   `env:"MINIO_USE_SSL"`

	KafkaBrokers []string `env:"KAFKA_BROKERS" envSeparator:"," envDefault:"localhost:9092"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	if err := runMigrations(cfg.DBDSN); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	storageClient, err := storage.NewClient(storage.Config{
		Endpoint:  cfg.MinioEndpoint,
		AccessKey: cfg.MinioAccessKey,
		SecretKey: cfg.MinioSecretKey,
		Bucket:    cfg.MinioBucket,
		UseSSL:    cfg.MinioUseSSL,
	})
	if err != nil {
		log.Fatalf("create storage client: %v", err)
	}
	if err := storageClient.EnsureBucket(ctx); err != nil {
		log.Fatalf("ensure storage bucket: %v", err)
	}

	authRepo := authinternal.NewRepository(pool)
	authService := authinternal.NewService(authRepo)
	authHandler := authinternal.NewHandler(authService)

	taskRepo := tasksinternal.NewRepository(pool)
	taskService := tasksinternal.NewService(taskRepo, storageClient)
	taskHandler := tasksinternal.NewHandler(taskService)

	if err := kafka.EnsureTopics(ctx, cfg.KafkaBrokers, kafka.TopicSubmissions, kafka.TopicResults); err != nil {
		log.Fatalf("ensure kafka topics: %v", err)
	}

	submissionsProducer := kafka.NewProducer(cfg.KafkaBrokers, kafka.TopicSubmissions)
	defer submissionsProducer.Close()

	submitRepo := submitinternal.NewRepository(pool)
	submitService := submitinternal.NewService(submitRepo, submissionsProducer)
	submitHandler := submitinternal.NewHandler(submitService)

	resultsConsumer := kafka.NewConsumer(cfg.KafkaBrokers, kafka.TopicResults, "server-results-consumer")
	defer resultsConsumer.Close()

	go resultsConsumer.Consume(ctx, func(ctx context.Context, data []byte) error {
		var msg kafka.ResponseMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[submit] invalid result message, skipping: %v", err)
			return nil
		}
		return submitService.HandleResult(ctx, msg)
	})

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
	}))

	r.Mount("/api/auth", authHandler.Routes())

	r.Group(func(r chi.Router) {
		r.Use(authinternal.AuthMiddleware(authService))
		r.Mount("/api/tasks", taskHandler.Routes())
		r.Mount("/api/submit", submitHandler.Routes())
	})

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
}

func runMigrations(dsn string) error {
	m, err := migrate.New("file://server/migrations", dsn)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
