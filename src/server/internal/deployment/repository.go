package deployment

import (
	"context"
	"database/sql"
)

type Repository interface {
	Create(ctx context.Context, f *Function) error
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
