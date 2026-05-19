package deployment

import (
	"context"
	"database/sql"
)

type Repository interface {
	Create(ctx context.Context, f *Function) error
	Exists(ctx context.Context, userID int, name string) (bool, error)
	ListByUserID(ctx context.Context, userID int) ([]Function, error)
	GetByName(ctx context.Context, userID int, name string) (*Function, error)
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
	query := `SELECT name, runtime, handler_name, module_name, timeout_seconds, memory_mb, created_at 
              FROM functions WHERE owner_id = $1 AND name = $2`

	f := &Function{}
	err := r.db.QueryRowContext(ctx, query, userID, name).Scan(
		&f.Name, &f.Runtime, &f.HandlerName, &f.ModuleName, &f.Timeout, &f.Memory, &f.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return f, nil
}
