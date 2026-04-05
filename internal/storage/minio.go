// internal/storage/minio.go
package storage

import (
	"context"
	"diploma-back/internal/config"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOClient struct {
	client        *minio.Client
	presignClient *minio.Client
	bucket        string
}

func NewMinIOClient(cfg *config.MinIOConfig) (*MinIOClient, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Separate client with public endpoint, used only to sign URLs for browser access.
	// Falls back to the internal endpoint if PUBLIC_ENDPOINT is not set.
	presignEndpoint := cfg.PublicEndpoint
	if presignEndpoint == "" {
		presignEndpoint = cfg.Endpoint
	}
	presignClient, err := minio.New(presignEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO presign client: %w", err)
	}

	// Create bucket if it doesn't exist
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinIOClient{
		client:        client,
		presignClient: presignClient,
		bucket:        cfg.Bucket,
	}, nil
}

// GetPresignedURL generates a presigned URL for downloading
func (m *MinIOClient) GetPresignedURL(ctx context.Context, objectName string) (string, error) {
	url, err := m.presignClient.PresignedGetObject(ctx, m.bucket, objectName, time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

func (m *MinIOClient) DownloadFile(objectName string) ([]byte, error) {
	obj, err := m.client.GetObject(context.Background(), m.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}
	return data, nil
}

func (m *MinIOClient) UploadFile(objectName string, fileStream io.Reader, size int64, contentType string) (info minio.UploadInfo, err error) {
	return m.client.PutObject(
		context.TODO(),
		m.bucket,
		objectName,
		fileStream, // <-- STREAM
		size,       // known size
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
}
