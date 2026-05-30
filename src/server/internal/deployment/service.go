package deployment

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service interface {
	Deploy(ctx context.Context, userID int, manifest DeployRequestManifest, artifactReader io.Reader) (string, error)
	ListByUserID(ctx context.Context, userID int) ([]Function, error)
	GetByName(ctx context.Context, userID int, name string) (*Function, error)
	Delete(ctx context.Context, userID int, name string) error
	ExecuteByAccessToken(ctx context.Context, rawToken string) (*ExecuteResponse, error)
}

type service struct {
	log          *zap.Logger
	repo         Repository
	storage      ArtifactStorage
	orchestrator OrchestratorClient
	tokenTTL     time.Duration
}

func NewService(storage ArtifactStorage, repo Repository, orchestrator OrchestratorClient, tokenTTL time.Duration, log *zap.Logger) Service {
	return &service{storage: storage, repo: repo, orchestrator: orchestrator, tokenTTL: tokenTTL, log: log}
}

func (s *service) Deploy(ctx context.Context, userID int, manifest DeployRequestManifest, artifactReader io.Reader) (string, error) {
	exists, err := s.repo.Exists(ctx, userID, manifest.Name)
	if err != nil {
		s.log.Error("Database check failed", zap.Error(err))
		return "", fmt.Errorf("database check failed")
	}
	if exists {
		return "", ErrFunctionAlreadyExists
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, artifactReader); err != nil {
		return "", fmt.Errorf("failed to buffer artifact")
	}
	artifactBytes := buf.Bytes()

	// Verification
	safe, reason, err := VerifyArtifact(ctx, artifactBytes, manifest)
	if err != nil {
		s.log.Error("Verifier unreachable", zap.Error(err))
		return "", fmt.Errorf("verification service unavailable")
	}
	if !safe {
		s.log.Warn("Artifact rejected", zap.String("reason", reason))
		return "", &ErrVerificationFailed{Reason: reason}
	}

	artifactHash, err := s.hashArtifact(bytes.NewReader(artifactBytes))
	if err != nil {
		return "", err
	}

	functionID := uuid.New()
	storageKey := fmt.Sprintf("functions/%d/%s.zip", userID, functionID.String())
	err = s.storage.Upload(ctx, storageKey, bytes.NewReader(artifactBytes))
	if err != nil {
		s.log.Error("Failed to upload artifact to storage", zap.Error(err))
		return "", fmt.Errorf("storage upload failed")
	}

	dbFunc := &Function{
		ID:           functionID,
		OwnerID:      userID,
		Name:         manifest.Name,
		Runtime:      manifest.Runtime,
		ModuleName:   manifest.Module,
		HandlerName:  manifest.Handler,
		ArtifactHash: artifactHash,
		Timeout:      manifest.Timeout,
		Memory:       manifest.Memory,
	}

	err = s.repo.Create(ctx, dbFunc)
	if err != nil {
		_ = s.storage.Delete(context.Background(), storageKey)
		s.log.Error("Failed to save function metadata", zap.Error(err))
		return "", fmt.Errorf("database error")
	}

	accessToken, err := s.generateAccessToken()
	if err != nil {
		s.log.Error("Failed to generate access token", zap.Error(err))
		return "", fmt.Errorf("access token generation failed")
	}

	accessTokenHash := s.hashToken(accessToken)
	expiresAt := time.Now().Add(s.tokenTTL)
	if err := s.repo.CreateAccessToken(ctx, functionID.String(), accessTokenHash, expiresAt); err != nil {
		s.log.Error("Failed to store access token", zap.Error(err))
		return "", fmt.Errorf("access token storage failed")
	}

	s.log.Info("Function successfully deployed", zap.String("function_id", functionID.String()))
	return accessToken, nil
}

func (s *service) ListByUserID(ctx context.Context, userID int) ([]Function, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *service) GetByName(ctx context.Context, userID int, name string) (*Function, error) {
	return s.repo.GetByName(ctx, userID, name)
}

func (s *service) Delete(ctx context.Context, userID int, name string) error {
	functionID, err := s.repo.Delete(ctx, userID, name)
	if err != nil {
		return err
	}

	storageKey := fmt.Sprintf("functions/%d/%s.zip", userID, functionID)
	err = s.storage.Delete(ctx, storageKey)
	if err != nil {
		s.log.Error("Failed to delete artifact from storage", zap.Error(err), zap.String("key", storageKey))
	}

	return nil
}

func (s *service) ExecuteByAccessToken(ctx context.Context, rawToken string) (*ExecuteResponse, error) {
	tokenHash := s.hashToken(rawToken)

	f, err := s.repo.GetByAccessToken(ctx, tokenHash)
	if err != nil {
		return nil, ErrAccessTokenInvalid
	}

	storageKey := fmt.Sprintf("functions/%d/%s.zip", f.OwnerID, f.ID)
	artifactReader, err := s.storage.Donwload(ctx, storageKey)
	if err != nil {
		s.log.Error("Failed to download artifact", zap.Error(err), zap.String("key", storageKey))
		return nil, fmt.Errorf("artifact download failed")
	}
	defer artifactReader.Close()

	artifactBytes, err := io.ReadAll(artifactReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact")
	}

	manifest := DeployRequestManifest{
		Name:    f.Name,
		Runtime: f.Runtime,
		Module:  f.ModuleName,
		Handler: f.HandlerName,
		Timeout: f.Timeout,
		Memory:  f.Memory,
	}

	result, err := s.orchestrator.Execute(ctx, manifest, artifactBytes)
	if err != nil {
		s.log.Error("Orchestrator execution failed", zap.Error(err))
		return nil, fmt.Errorf("orchestrator execution failed")
	}

	return result, nil
}

func (s *service) hashArtifact(tee io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, tee); err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (s *service) generateAccessToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "oblak_exec_" + hex.EncodeToString(bytes), nil
}

func (s *service) hashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}
