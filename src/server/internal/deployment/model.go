package deployment

import (
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

type ExecuteResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
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
