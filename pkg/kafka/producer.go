package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}

	return &Producer{
		writer: writer,
	}
}

func (p *Producer) Send(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: data,
	}

	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		return err
	}

	log.Printf("[Kafka] Sent: key=%s", key)
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
