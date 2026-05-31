package deployment

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, f *Function) error
	Exists(ctx context.Context, userID int, name string) (bool, error)
	ListByUserID(ctx context.Context, userID int) ([]Function, error)
	GetByName(ctx context.Context, userID int, name string) (*Function, error)
	Delete(ctx context.Context, userID int, name string) (string, error)
	CreateAccessToken(ctx context.Context, functionID string, tokenHash string, expiresAt time.Time) error
	DeleteAccessTokensByFunctionID(ctx context.Context, functionID uuid.UUID) error
	GetByAccessToken(ctx context.Context, tokenHash string) (*Function, error)
}

type sqlRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &sqlRepository{db: db}
}

func (r *sqlRepository) Create(ctx context.Context, f *Function) error {
	const query = `
		INSERT INTO functions (id, owner_id, name, runtime, module_name, handler_name, artifact_hash, timeout_seconds, memory_mb, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`

	_, err := r.db.ExecContext(ctx, query,
		f.ID, f.OwnerID, f.Name, f.Runtime, f.ModuleName, f.HandlerName, f.ArtifactHash, f.Timeout, f.Memory)

	return err
}

func (r *sqlRepository) Exists(ctx context.Context, userID int, name string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM functions WHERE owner_id = $1 AND name = $2)`
	err := r.db.QueryRowContext(ctx, query, userID, name).Scan(&exists)
	return exists, err
}

func (r *sqlRepository) ListByUserID(ctx context.Context, userID int) ([]Function, error) {
	query := `SELECT name, runtime FROM functions WHERE owner_id = $1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var funcs []Function
	for rows.Next() {
		var f Function
		if err := rows.Scan(&f.Name, &f.Runtime); err != nil {
			return nil, err
		}
		funcs = append(funcs, f)
	}
	return funcs, nil
}

func (r *sqlRepository) GetByName(ctx context.Context, userID int, name string) (*Function, error) {
	query := `SELECT id, name, runtime, handler_name, module_name, timeout_seconds, memory_mb, created_at 
              FROM functions WHERE owner_id = $1 AND name = $2`

	f := &Function{}
	err := r.db.QueryRowContext(ctx, query, userID, name).Scan(
		&f.ID, &f.Name, &f.Runtime, &f.HandlerName, &f.ModuleName, &f.Timeout, &f.Memory, &f.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (r *sqlRepository) Delete(ctx context.Context, userID int, name string) (string, error) {
	var functionID string
	querySelect := `SELECT id FROM functions WHERE owner_id = $1 AND name = $2`
	err := r.db.QueryRowContext(ctx, querySelect, userID, name).Scan(&functionID)
	if err != nil {
		return "", err
	}

	queryDelete := `DELETE FROM functions WHERE owner_id = $1 AND name = $2`
	_, err = r.db.ExecContext(ctx, queryDelete, userID, name)
	if err != nil {
		return "", err
	}

	return functionID, nil
}

func (r *sqlRepository) CreateAccessToken(ctx context.Context, functionID string, tokenHash string, expiresAt time.Time) error {
	query := `INSERT INTO function_access_tokens (function_id, token_hash, expires_at) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, functionID, tokenHash, expiresAt)
	return err
}

func (r *sqlRepository) DeleteAccessTokensByFunctionID(ctx context.Context, functionID uuid.UUID) error {
	query := `DELETE FROM function_access_tokens WHERE function_id = $1`
	_, err := r.db.ExecContext(ctx, query, functionID)
	return err
}

func (r *sqlRepository) GetByAccessToken(ctx context.Context, tokenHash string) (*Function, error) {
	query := `
		SELECT f.id, f.owner_id, f.name, f.runtime, f.module_name, f.handler_name,
			f.artifact_hash, f.timeout_seconds, f.memory_mb, f.created_at
		FROM function_access_tokens t
		JOIN functions f ON f.id = t.function_id
		WHERE t.token_hash = $1 AND t.expires_at > NOW()
	`

	f := &Function{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&f.ID, &f.OwnerID, &f.Name, &f.Runtime, &f.ModuleName, &f.HandlerName,
		&f.ArtifactHash, &f.Timeout, &f.Memory, &f.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return f, nil
}
