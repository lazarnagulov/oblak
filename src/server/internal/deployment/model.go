package deployment

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Function represents the core database entity for a deployed serverless function.
type Function struct {
	ID           uuid.UUID `json:"id" db:"id" example:"9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"`
	OwnerID      int       `json:"owner_id" db:"owner_id" example:"1"`
	Name         string    `json:"name" db:"name" example:"my-api-handler"`
	Runtime      string    `json:"runtime" db:"runtime" example:"python3.10"`
	ModuleName   string    `json:"module_name" db:"module_name" example:"main"`
	HandlerName  string    `json:"handler_name" db:"handler_name" example:"handler"`
	ArtifactHash string    `json:"artifact_hash" db:"artifcat_hash" example:"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`
	Timeout      int       `json:"timeout_seconds" db:"timeout_seconds" example:"10"`
	Memory       int       `json:"memory_mb" db:"memory_mb" example:"128"`
	CreatedAt    time.Time `json:"created_at" db:"created_at" example:"2026-05-22T12:00:00Z"`
}

// FunctionResponse represents the detailed view of a function returned to the client.
// @Name FunctionResponse
type FunctionResponse struct {
	ID          string    `json:"id" example:"9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"`
	Name        string    `json:"name" example:"my-api-handler"`
	Runtime     string    `json:"runtime" example:"python3.10"`
	ModuleName  string    `json:"module" example:"main"`
	HandlerName string    `json:"handler" example:"handler"`
	Timeout     int       `json:"timeout" example:"10"`
	Memory      int       `json:"memory" example:"128"`
	CreatedAt   time.Time `json:"created_at" example:"2026-05-22T12:00:00Z"`
}

// DeployRequestManifest represents the metadata required to deploy a new serverless function.
// @Name DeployRequestManifest
type DeployRequestManifest struct {
	// Unique name of the function
	Name string `json:"name" binding:"required" example:"my-api-handler"`
	// Target execution environment runtime
	Runtime string `json:"runtime" binding:"required" example:"python3.10"`
	// Entrypoint module/file name
	Module string `json:"module" binding:"required" example:"main"`
	// Function name inside the module to invoke
	Handler string `json:"handler" binding:"required" example:"handler"`
	// Maximum execution time in seconds
	// minimum: 1
	// maximum: 30
	Timeout int `json:"timeout" binding:"required,min=1,max=30" example:"10"`
	// Memory allocation size in Megabytes
	// minimum: 64
	// maximum: 512
	Memory int `json:"memory" binding:"required,min=64,max=512" example:"128"`
}

// FunctionListResponse represents the lightweight summary of a function for list views.
// @Name FunctionListResponse
type FunctionListResponse struct {
	Name    string `json:"name" example:"my-api-handler"`
	Runtime string `json:"runtime" example:"python3.10"`
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
