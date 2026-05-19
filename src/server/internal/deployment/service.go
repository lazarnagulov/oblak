package deployment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service interface {
	Deploy(ctx context.Context, userID int, manifest DeployRequestManifest, artifactReader io.Reader) error
}

type service struct {
	log     *zap.Logger
	repo    Repository
	storage ArtifactStorage
}

func NewService(storage ArtifactStorage, repo Repository, log *zap.Logger) Service {
	return &service{storage: storage, repo: repo, log: log}
}

func (s *service) Deploy(ctx context.Context, userID int, manifest DeployRequestManifest, artifactReader io.Reader) error {
	functionID := uuid.New()

	var buf bytes.Buffer
	tee := io.TeeReader(artifactReader, &buf)
	artifactHash, err := s.hashArtifact(tee)
	if err != nil {
		return err
	}

	storageKey := fmt.Sprintf("functions/%d/%s.zip", userID, functionID.String())
	err = s.storage.Upload(ctx, storageKey, &buf)
	if err != nil {
		s.log.Error("Failed to upload artifact to storage", zap.Error(err))
		return fmt.Errorf("storage upload failed")
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
		return fmt.Errorf("database error")
	}

	s.log.Info("Function successfully deployed", zap.String("function_id", functionID.String()))
	return nil
}

func (s *service) hashArtifact(tee io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, tee); err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
