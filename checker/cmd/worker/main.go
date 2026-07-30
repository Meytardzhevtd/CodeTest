package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v11"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/meytardzhevtd/CodeTest/checker/internal/grpc/judgepb"
	"github.com/meytardzhevtd/CodeTest/checker/internal/worker"
)

type config struct {
	CoordinatorAddr string `env:"COORDINATOR_ADDR" envDefault:"localhost:50051"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, err := grpc.NewClient(cfg.CoordinatorAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial coordinator at %s: %v", cfg.CoordinatorAddr, err)
	}
	defer conn.Close()

	id := uuid.NewString()[:8]
	w := worker.New(id, judgepb.NewCoordinatorClient(conn))

	log.Printf("[worker %s] подключён к координатору %s", id, cfg.CoordinatorAddr)
	w.Run(ctx)
}
