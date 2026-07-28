package kafka

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.LastOffset,
		MinBytes:    10e3,
		MaxBytes:    10e6,
		MaxWait:     1 * time.Second,
	})

	return &Consumer{
		reader: reader,
	}
}

func (c *Consumer) Consume(ctx context.Context, handler func(ctx context.Context, data []byte) error) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("[Kafka] Consumer stopped")
			return
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("[Kafka] Read error: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if err := handler(ctx, msg.Value); err != nil {
				log.Printf("[Kafka] Handler error: %v", err)
				continue // не коммитим оффсет — сообщение будет доставлено повторно
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("[Kafka] Commit error: %v", err)
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
