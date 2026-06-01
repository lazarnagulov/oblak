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
	CreateExecutionRecord(ctx context.Context, record *ExecutionRecord) (*ExecutionRecord, error)
	UpdateExecutionRecord(ctx context.Context, record *ExecutionRecord) error
	GetExecutionsByFunctionID(ctx context.Context, functionID uuid.UUID) ([]ExecutionRecord, error)
	GetExecutionByID(ctx context.Context, executionID int64, userID int) (*ExecutionRecord, error)
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
	query := `SELECT id, owner_id, name, runtime, handler_name, module_name, timeout_seconds, memory_mb, created_at 
              FROM functions WHERE owner_id = $1 AND name = $2`

	f := &Function{}
	err := r.db.QueryRowContext(ctx, query, userID, name).Scan(
		&f.ID, &f.OwnerID, &f.Name, &f.Runtime, &f.HandlerName, &f.ModuleName, &f.Timeout, &f.Memory, &f.CreatedAt,
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

func (r *sqlRepository) CreateExecutionRecord(ctx context.Context, record *ExecutionRecord) (*ExecutionRecord, error) {
	query := `
		INSERT INTO executions (function_id, status, started_at, finished_at, execution_time_ms, logs, result_data, error_message, worker_node, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query,
		record.FunctionID, record.Status, record.StartedAt, record.FinishedAt,
		record.ExecutionTimeMs, record.Logs, record.ResultData, record.ErrorMessage, record.WorkerNode,
	).Scan(&record.ID)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (r *sqlRepository) UpdateExecutionRecord(ctx context.Context, record *ExecutionRecord) error {
	query := `
		UPDATE executions
		SET status = $1, finished_at = $2, execution_time_ms = $3, logs = $4, result_data = $5, error_message = $6, worker_node = $7
		WHERE id = $8
	`
	_, err := r.db.ExecContext(ctx, query,
		record.Status, record.FinishedAt, record.ExecutionTimeMs, record.Logs,
		record.ResultData, record.ErrorMessage, record.WorkerNode, record.ID,
	)
	return err
}

func (r *sqlRepository) GetExecutionsByFunctionID(ctx context.Context, functionID uuid.UUID) ([]ExecutionRecord, error) {
	query := `
		SELECT id, function_id, status, started_at, finished_at, execution_time_ms, logs, result_data, error_message, worker_node
		FROM executions
		WHERE function_id = $1
		ORDER BY started_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, functionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ExecutionRecord
	for rows.Next() {
		record, err := scanExecutionRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}
	return records, nil
}

func (r *sqlRepository) GetExecutionByID(ctx context.Context, executionID int64, userID int) (*ExecutionRecord, error) {
	query := `
		SELECT
			e.id,
			e.function_id,
			e.status,
			e.started_at,
			e.finished_at,
			e.execution_time_ms,
			e.logs,
			e.result_data,
			e.error_message,
			e.worker_node
		FROM executions e
		JOIN functions f ON e.function_id = f.id
		WHERE e.id = $1
		AND f.owner_id = $2`

	record, err := scanExecutionRecord(r.db.QueryRowContext(ctx, query, executionID, userID))
	if err != nil {
		return nil, err
	}
	return record, nil
}

func scanExecutionRecord(scanner interface{ Scan(dest ...any) error }) (*ExecutionRecord, error) {
	record := &ExecutionRecord{}

	var startedAt, finishedAt sql.NullTime
	var logs, resultData, errorMessage, workerNode sql.NullString

	err := scanner.Scan(
		&record.ID,
		&record.FunctionID,
		&record.Status,
		&startedAt,
		&finishedAt,
		&record.ExecutionTimeMs,
		&logs,
		&resultData,
		&errorMessage,
		&workerNode,
	)
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		record.StartedAt = &startedAt.Time
	}

	if finishedAt.Valid {
		record.FinishedAt = &finishedAt.Time
	}

	record.Logs = logs.String
	record.ResultData = resultData.String
	record.ErrorMessage = errorMessage.String
	record.WorkerNode = workerNode.String

	return record, nil
}
