package deployment

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DeployRequestManifest struct {
	Name    string `json:"name" binding:"required"`
	Runtime string `json:"runtime" binding:"required"`
	Module  string `json:"module" binding:"required"`
	Handler string `json:"handler" binding:"required"`
	Timeout int    `json:"timeout" binding:"required,min=1,max=30"`
	Memory  int    `json:"memory" binding:"required,min=64,max=512"`
}

type Function struct {
	ID           uuid.UUID `json:"id" db:"id"`
	OwnerID      int       `json:"owner_id" db:"owner_id"`
	Name         string    `json:"name" db:"name"`
	Runtime      string    `json:"runtime" db:"runtime"`
	ModuleName   string    `json:"module_name" db:"module_name"`
	HandlerName  string    `json:"handler_name" db:"handler_name"`
	ArtifactHash string    `json:"artifact_hash" db:"artifcat_hash"`
	Timeout      int       `json:"timeout_seconds" db:"timeout_seconds"`
	Memory       int       `json:"memory_mb" db:"memory_mb"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type FunctionResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Runtime     string    `json:"runtime"`
	ModuleName  string    `json:"module"`
	HandlerName string    `json:"handler"`
	Timeout     int       `json:"timeout"`
	Memory      int       `json:"memory"`
	CreatedAt   time.Time `json:"created_at"`
}

type FunctionListResponse struct {
	Name    string `json:"name"`
	Runtime string `json:"runtime"`
}

type ExecuteResult struct {
	ID              int64           `json:"id"`
	Status          ExecutionStatus `json:"status"`
	Logs            string          `json:"logs"`
	ErrorMessage    string          `json:"error_message"`
	Result          any             `json:"result"`
	ExecutionTimeMs int64           `json:"execution_time_ms"`
	WorkerNode      string          `json:"worker_node"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	FinishedAt      *time.Time      `json:"finished_at,omitempty"`
}

type ExecuteResponse struct {
	ID              int64           `json:"id"`
	Status          ExecutionStatus `json:"status"`
	Logs            string          `json:"logs"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	Result          any             `json:"result"`
	ExecutionTimeMs int64           `json:"execution_time_ms"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	FinishedAt      *time.Time      `json:"finished_at,omitempty"`
}

type ExecutionStatus string

const (
	ExecutionStatusPending ExecutionStatus = "PENDING"
	ExecutionStatusRunning ExecutionStatus = "RUNNING"
	ExecutionStatusSuccess ExecutionStatus = "SUCCESS"
	ExecutionStatusFailed  ExecutionStatus = "FAILED"
	ExecutionStatusTimeout ExecutionStatus = "TIMEOUT"
)

type ExecutionRecord struct {
	ID              int64           `json:"id" db:"id"`
	FunctionID      uuid.UUID       `json:"function_id" db:"function_id"`
	Status          ExecutionStatus `json:"status" db:"status"`
	StartedAt       *time.Time      `json:"started_at" db:"started_at"`
	FinishedAt      *time.Time      `json:"finished_at" db:"finished_at"`
	ExecutionTimeMs int64           `json:"execution_time_ms" db:"execution_time_ms"`
	Logs            string          `json:"logs" db:"logs"`
	ResultData      string          `json:"result_data" db:"result_data"`
	ErrorMessage    string          `json:"error_message" db:"error_message"`
	WorkerNode      string          `json:"worker_node" db:"worker_node"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

type ExecutionRecordResponse struct {
	ID              int64           `json:"id"`
	Status          ExecutionStatus `json:"status"`
	StartedAt       *time.Time      `json:"started_at"`
	FinishedAt      *time.Time      `json:"finished_at"`
	ExecutionTimeMs int64           `json:"execution_time_ms"`
	Logs            string          `json:"logs"`
	Result          any             `json:"result"`
	ErrorMessage    string          `json:"error_message,omitempty"`
}

type ExecutionRecordSummary struct {
	ID              int64           `json:"id"`
	Status          ExecutionStatus `json:"status"`
	StartedAt       *time.Time      `json:"started_at"`
	FinishedAt      *time.Time      `json:"finished_at"`
	ExecutionTimeMs int64           `json:"execution_time_ms"`
}

func ToListResponse(f *Function) FunctionListResponse {
	return FunctionListResponse{
		Name:    f.Name,
		Runtime: f.Runtime,
	}
}

func ToResponse(f *Function) FunctionResponse {
	return FunctionResponse{
		ID:          f.ID.String(),
		Name:        f.Name,
		Runtime:     f.Runtime,
		ModuleName:  f.ModuleName,
		HandlerName: f.HandlerName,
		Timeout:     f.Timeout,
		Memory:      f.Memory,
		CreatedAt:   f.CreatedAt,
	}
}

func ToExecutionRecordSummary(r *ExecutionRecord) ExecutionRecordSummary {
	return ExecutionRecordSummary{
		ID:              r.ID,
		Status:          r.Status,
		StartedAt:       r.StartedAt,
		FinishedAt:      r.FinishedAt,
		ExecutionTimeMs: r.ExecutionTimeMs,
	}
}

func ToExecutionRecordResponse(r *ExecutionRecord) ExecutionRecordResponse {
	result := any(nil)
	if r.ResultData != "" {
		var jsonData any
		if err := json.Unmarshal([]byte(r.ResultData), &jsonData); err == nil {
			result = jsonData
		} else {
			result = r.ResultData
		}
	}

	return ExecutionRecordResponse{
		ID:              r.ID,
		Status:          r.Status,
		StartedAt:       r.StartedAt,
		FinishedAt:      r.FinishedAt,
		ExecutionTimeMs: r.ExecutionTimeMs,
		Logs:            r.Logs,
		Result:          result,
		ErrorMessage:    r.ErrorMessage,
	}
}
