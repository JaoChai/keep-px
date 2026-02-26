package service

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"github.com/jaochai/pixlinks/backend/internal/config"
)

type StorageService struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewStorageService(cfg *config.Config) *StorageService {
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(cfg.S3Endpoint),
		Region:       "auto",
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, ""),
	})

	return &StorageService{
		client:    client,
		bucket:    cfg.S3Bucket,
		publicURL: cfg.S3PublicURL,
	}
}

func (s *StorageService) Upload(ctx context.Context, file io.Reader, originalFilename string, contentType string) (string, error) {
	ext := path.Ext(originalFilename)
	key := fmt.Sprintf("uploads/%s%s", uuid.New().String(), ext)

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	url := fmt.Sprintf("%s/%s", s.publicURL, key)
	return url, nil
}

func (s *StorageService) IsConfigured() bool {
	return s.bucket != "" && s.publicURL != ""
}
