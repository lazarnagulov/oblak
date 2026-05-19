package deployment

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

type ArtifactStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader) error
	Donwload(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type MinioConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	UseSSL     bool
}

type minioStorage struct {
	client     *minio.Client
	bucketName string
}

func NewMinioStorage(cfg MinioConfig, log *zap.Logger) (ArtifactStorage, error) {
	log.Info("Connecting to MinIO",
		zap.String("endpoint", cfg.Endpoint),
		zap.String("bucket", cfg.BucketName),
		zap.Bool("useSSL", cfg.UseSSL),
	)

	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		log.Error("Failed to initialize MinIO client",
			zap.Error(err),
		)
		return nil, fmt.Errorf("Failed to initialize minio client: %w", err)
	}

	storage := &minioStorage{
		client:     minioClient,
		bucketName: cfg.BucketName,
	}

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		log.Info("creating MinIO bucket",
			zap.String("bucket", cfg.BucketName),
		)
		err = minioClient.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Error("failed to create MinIO bucket",
				zap.String("bucket", cfg.BucketName),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to create minio bucket: %w", err)
		}

		log.Info("MinIO bucket created",
			zap.String("bucket", cfg.BucketName),
		)
	}

	log.Info("Successfully connected to MinIO",
		zap.String("endpoint", cfg.Endpoint),
		zap.String("bucket", cfg.BucketName),
	)

	return storage, nil
}

func (m *minioStorage) Upload(ctx context.Context, key string, reader io.Reader) error {
	_, err := m.client.PutObject(ctx, m.bucketName, key, reader, -1, minio.PutObjectOptions{
		ContentType: "application/zip",
	})
	if err != nil {
		return fmt.Errorf("minio upload error for key %s: %w", key, err)
	}
	return nil
}

func (m *minioStorage) Donwload(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := m.client.GetObject(ctx, m.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio download error for key %s: %w", key, err)
	}

	_, err = object.Stat()
	if err != nil {
		object.Close()
		return nil, fmt.Errorf("failed to find object %s: %w", key, err)
	}

	return object, nil
}

func (m *minioStorage) Delete(ctx context.Context, key string) error {
	err := m.client.RemoveObject(ctx, m.bucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete minio object %s: %w", key, err)
	}
	return nil
}

func (m *minioStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := m.client.StatObject(ctx, m.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence of %s: %w", key, err)
	}
	return true, nil
}
