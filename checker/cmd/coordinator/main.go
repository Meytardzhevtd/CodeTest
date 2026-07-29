package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/meytardzhevtd/CodeTest/pkg/kafka"
)

type config struct {
	KafkaBrokers []string `env:"KAFKA_BROKERS" envSeparator:"," envDefault:"localhost:9092"`
	GroupID      string   `env:"KAFKA_GROUP_ID" envDefault:"coordinator"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	producer := kafka.NewProducer(cfg.KafkaBrokers, kafka.TopicResults)
	defer producer.Close()

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, kafka.TopicSubmissions, cfg.GroupID)
	defer consumer.Close()

	log.Printf("[coordinator] listening on topic %q, group %q", kafka.TopicSubmissions, cfg.GroupID)
	consumer.Consume(ctx, handleSubmission(producer))
}

// handleSubmission разбирает сообщение о сабмишне и публикует вердикт в топик результатов.
// Возврат ошибки означает, что оффсет не будет закоммичен и сообщение придёт снова —
// поэтому ошибками считаются только сбои, после которых имеет смысл повторить попытку
// (например, недоступность брокера при публикации результата). Сами по себе "плохие"
// входные данные (невалидный JSON, пустой submission_id) не ретраятся: это не поможет,
// такое сообщение уже никогда не станет валидным.
func handleSubmission(producer *kafka.Producer) func(ctx context.Context, data []byte) error {
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

		start := time.Now()
		log.Printf("[coordinator] submission %s: начал обработку (task=%s, language=%s, code=%d bytes)",
			msg.SubmissionID, msg.TaskID, msg.Language, len(msg.Code))

		result := judge(msg)
		log.Printf("[coordinator] submission %s: вердикт — %s", result.SubmissionID, result.Status)
		if result.Error != "" {
			log.Printf("[coordinator] submission %s: причина — %s", result.SubmissionID, result.Error)
		}

		if err := producer.Send(ctx, result.SubmissionID, result); err != nil {
			log.Printf("[coordinator] submission %s: не удалось опубликовать результат: %v", msg.SubmissionID, err)
			return err
		}

		log.Printf("[coordinator] submission %s: обработка завершена за %s", result.SubmissionID, time.Since(start))
		return nil
	}
}

// judge — временная заглушка вместо реального запуска решения в изолированном контейнере
// (см. обсуждение RunInContainer/LanguageConfig). Единственная содержательная проверка
// сейчас — известен ли язык; для всех остальных случаев вердикт всегда "OK".
func judge(msg kafka.SubmissionMessage) kafka.ResponseMessage {
	switch msg.Language {
	case "python", "go", "cpp", "javascript":
		// TODO: поднять контейнер по LanguageConfig, скомпилировать (если нужно),
		// прогнать все test cases и вернуть настоящий вердикт (OK/WA/RE/CE/TL/ML).
		return kafka.ResponseMessage{
			SubmissionID: msg.SubmissionID,
			Status:       "OK",
		}
	default:
		return kafka.ResponseMessage{
			SubmissionID: msg.SubmissionID,
			Status:       "CE",
			Error:        fmt.Sprintf("unsupported language: %s", msg.Language),
		}
	}
}
