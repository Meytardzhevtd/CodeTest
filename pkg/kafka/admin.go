package kafka

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/segmentio/kafka-go"
)

func EnsureTopics(ctx context.Context, brokers []string, topics ...string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("no kafka brokers configured")
	}

	conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("dial kafka: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Printf("failed to close kafka connection: %v\n", err)
		}
	}()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("get controller: %w", err)
	}

	controllerConn, err := kafka.DialContext(ctx, "tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return fmt.Errorf("dial controller: %w", err)
	}
	defer func() {
		if err := controllerConn.Close(); err != nil {
			fmt.Printf("failed to close controller connection: %v\n", err)
		}
	}()

	configs := make([]kafka.TopicConfig, len(topics))
	for i, topic := range topics {
		configs[i] = kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}
	}

	if err := controllerConn.CreateTopics(configs...); err != nil {
		return fmt.Errorf("create topics: %w", err)
	}
	return nil
}
