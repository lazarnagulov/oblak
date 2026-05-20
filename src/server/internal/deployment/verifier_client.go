package deployment

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"runtime"
	"time"
)

const verifierSocketPath = "/tmp/oblak_verifier.sock"

type verifierRequest struct {
	ArtifactB64 string                `json:"artifact_b64"`
	Manifest    DeployRequestManifest `json:"manifest"`
}

type verifierResponse struct {
	Safe   bool   `json:"safe"`
	Reason string `json:"reason"`
}

func VerifyArtifact(ctx context.Context, artifactBytes []byte, manifest DeployRequestManifest) (bool, string, error) {
	dialer := net.Dialer{Timeout: 5 * time.Second}

	var conn net.Conn
	var err error
	if runtime.GOOS == "linux" {
		conn, err = dialer.DialContext(ctx, "unix", verifierSocketPath)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", "127.0.0.1:9876")
	}

	if err != nil {
		return false, "", fmt.Errorf("could not connect to verifier: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(60 * time.Second))

	req := verifierRequest{
		ArtifactB64: base64.StdEncoding.EncodeToString(artifactBytes),
		Manifest:    manifest,
	}
	if err := writeMessage(conn, req); err != nil {
		return false, "", fmt.Errorf("failed to send to verifier: %w", err)
	}

	var resp verifierResponse
	if err := readMessage(conn, &resp); err != nil {
		return false, "", fmt.Errorf("failed to read verifier response: %w", err)
	}

	return resp.Safe, resp.Reason, nil
}

func writeMessage(w io.Writer, v any) error {
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

func readMessage(r io.Reader, v any) error {
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
