package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateUser(ctx context.Context, email, username, passwordHash string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, email, username, created_at
	`, email, username, passwordHash).Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err, "users_email_key") {
			return User{}, ErrEmailTaken
		}
		if isUniqueViolation(err, "users_username_key") {
			return User{}, ErrUsernameTaken
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, string, error) {
	var user User
	var passwordHash string
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, username, password_hash, created_at
		FROM users
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Username, &passwordHash, &user.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", fmt.Errorf("get user by email: %w", err)
	}
	return user, passwordHash, nil
}

func isUniqueViolation(err error, constraint string) bool {
	if err == nil {
		return false
	}
	return err.Error() == fmt.Sprintf("ERROR: duplicate key value violates unique constraint \"%s\" (SQLSTATE 23505)", constraint)
}
