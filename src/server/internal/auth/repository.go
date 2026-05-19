package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Repository interface {
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	CreateAPIKey(ctx context.Context, apiKey *APIKey) error
	FindUserIDByHash(ctx context.Context, hash string) (int, error)
	DeleteAPIKeyByHash(ctx context.Context, hash string) error
}

type sqlRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &sqlRepository{db: db}
}

func (r *sqlRepository) CreateAPIKey(ctx context.Context, apiKey *APIKey) error {
	query := `INSERT INTO api_keys (user_id, key_hash, name, expires_at) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query, apiKey.UserID, apiKey.KeyHash, apiKey.Name, apiKey.ExpiresAt).
		Scan(&apiKey.ID, &apiKey.CreatedAt)
}

func (r *sqlRepository) DeleteAPIKeyByHash(ctx context.Context, hash string) error {
	query := `DELETE FROM api_keys WHERE key_hash = $1`
	_, err := r.db.ExecContext(ctx, query, hash)
	return err
}

func (r *sqlRepository) FindUserIDByHash(ctx context.Context, hash string) (int, error) {
	query := `SELECT user_id FROM api_keys WHERE key_hash = $1 AND expires_at > $2`
	var userID int
	err := r.db.QueryRowContext(ctx, query, hash, time.Now()).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.New("invalid or expired token")
		}
		return 0, err
	}
	return userID, nil
}

func (r *sqlRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	query := `SELECT id, username, password_hash, created_at FROM users WHERE username = $1`

	var user User
	err := r.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}
