package deployment

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

type OrchestratorClient interface {
	Execute(ctx context.Context, manifest DeployRequestManifest, artifactBytes []byte, payload []byte) (*ExecuteResult, error)
}

type OrchestratorConfig struct {
	SocketPath string
	Timeout    time.Duration
}

type orchestratorClient struct {
	socketPath string
	timeout    time.Duration
}

func NewOrchestratorClient(cfg OrchestratorConfig) OrchestratorClient {
	return &orchestratorClient{socketPath: cfg.SocketPath, timeout: cfg.Timeout}
}

type orchestratorRequest struct {
	ArtifactB64 string                `json:"artifact_b64"`
	Manifest    DeployRequestManifest `json:"manifest"`
	Payload     string                `json:"payload"`
}

type orchestratorResponse struct {
	Status          ExecutionStatus `json:"status"`
	Logs            string          `json:"logs"`
	Result          any             `json:"result"`
	ExecutionTimeMs int64           `json:"execution_time_ms"`
	ErrorMessage    string          `json:"error_message"`
	WorkerNode      string          `json:"worker_node"`
}

func (o *orchestratorClient) Execute(ctx context.Context, manifest DeployRequestManifest, artifactBytes []byte, payload []byte) (*ExecuteResult, error) {
	dialer := net.Dialer{Timeout: o.timeout + time.Second*time.Duration(manifest.Timeout)}
	conn, err := dialer.DialContext(ctx, "unix", o.socketPath)
	if err != nil {
		return nil, fmt.Errorf("could not connect to orchestrator: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(o.timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetDeadline(deadline)

	request := orchestratorRequest{
		ArtifactB64: base64.StdEncoding.EncodeToString(artifactBytes),
		Manifest:    manifest,
		Payload:     string(payload),
	}
	if err := writeOrchestratorMessage(conn, request); err != nil {
		return nil, fmt.Errorf("failed to send request to orchestrator: %w", err)
	}

	var resp orchestratorResponse
	if err := readOrchestratorMessage(conn, &resp); err != nil {
		return nil, fmt.Errorf("failed to read orchestrator response: %w", err)
	}

	return &ExecuteResult{
		Status: resp.Status, Logs: resp.Logs, ErrorMessage: resp.ErrorMessage, Result: resp.Result, ExecutionTimeMs: resp.ExecutionTimeMs, WorkerNode: resp.WorkerNode}, nil
}

func writeOrchestratorMessage(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	length := uint32(len(data))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func readOrchestratorMessage(r io.Reader, v any) error {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return err
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
