package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	authinternal "github.com/meytardzhevtd/CodeTest/server/internal/auth"
)

type config struct {
	DBDSN            string `env:"DB_DSN,required"`
	JWTAccessSecret  string `env:"JWT_ACCESS_SECRET,required"`
	JWTRefreshSecret string `env:"JWT_REFRESH_SECRET,required"`
	JWTIssuer        string `env:"JWT_ISSUER"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	ctx := context.Background()
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

	repo := authinternal.NewRepository(pool)
	service := authinternal.NewService(repo)
	handler := authinternal.NewHandler(service)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}, AllowCredentials: true}))

	r.Mount("/api/auth", handler.Routes())

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("serve: %v", err)
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
