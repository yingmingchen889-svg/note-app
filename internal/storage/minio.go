package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/user/note-app/internal/config"
)

type MinIOClient struct {
	client *minio.Client
	bucket string
}

func NewMinIOClient(cfg config.MinIOConfig) (*MinIOClient, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return &MinIOClient{client: client, bucket: cfg.Bucket}, nil
}

type PresignResult struct {
	URL       string `json:"url"`
	ObjectKey string `json:"object_key"`
}

func (m *MinIOClient) Presign(ctx context.Context, contentType string) (*PresignResult, error) {
	objectKey := fmt.Sprintf("uploads/%s/%s", time.Now().Format("2006/01/02"), uuid.New().String())

	presignedURL, err := m.client.PresignedPutObject(ctx, m.bucket, objectKey, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("presign: %w", err)
	}

	return &PresignResult{
		URL:       presignedURL.String(),
		ObjectKey: objectKey,
	}, nil
}

func (m *MinIOClient) ObjectURL(objectKey string) string {
	u := &url.URL{
		Scheme: "http",
		Host:   m.client.EndpointURL().Host,
		Path:   fmt.Sprintf("/%s/%s", m.bucket, objectKey),
	}
	return u.String()
}
