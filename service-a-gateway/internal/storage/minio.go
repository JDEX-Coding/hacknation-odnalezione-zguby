package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
)

// MinIOStorage handles image uploads to MinIO
type MinIOStorage struct {
	client         *minio.Client
	bucketName     string
	endpoint       string
	publicEndpoint string
	useSSL         bool
}

// NewMinIOStorage creates a new MinIO storage client
func NewMinIOStorage(endpoint, publicEndpoint, accessKey, secretKey, bucketName string, useSSL bool) (*MinIOStorage, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// If publicEndpoint is empty, fallback to endpoint
	if publicEndpoint == "" {
		publicEndpoint = endpoint
	}

	storage := &MinIOStorage{
		client:         minioClient,
		bucketName:     bucketName,
		endpoint:       endpoint,
		publicEndpoint: publicEndpoint,
		useSSL:         useSSL,
	}

	// Verify bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("bucket '%s' does not exist", bucketName)
	}

	log.Info().
		Str("endpoint", endpoint).
		Str("public_endpoint", publicEndpoint).
		Str("bucket", bucketName).
		Msg("MinIO storage initialized")

	return storage, nil
}

// UploadImage uploads an image to MinIO and returns the public URL
func (s *MinIOStorage) UploadImage(ctx context.Context, reader io.Reader, filename string, contentType string, size int64) (string, error) {
	// Generate unique filename
	ext := filepath.Ext(filename)
	uniqueFilename := fmt.Sprintf("uploads/%s/%s%s", time.Now().Format("2006-01-02"), uuid.New().String(), ext)

	// Upload to MinIO
	_, err := s.client.PutObject(
		ctx,
		s.bucketName,
		uniqueFilename,
		reader,
		size,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	// Generate public URL
	protocol := "http"
	if s.useSSL {
		protocol = "https"
	}
	publicURL := fmt.Sprintf("%s://%s/%s/%s", protocol, s.publicEndpoint, s.bucketName, uniqueFilename)

	log.Info().
		Str("filename", filename).
		Str("unique_filename", uniqueFilename).
		Str("url", publicURL).
		Msg("Image uploaded successfully")

	return publicURL, nil
}

// DeleteImage deletes an image from MinIO
func (s *MinIOStorage) DeleteImage(ctx context.Context, imageURL string) error {
	// Extract object key from URL
	// URL format: http://localhost:9000/bucket-name/path/to/file.jpg
	// We need to extract "path/to/file.jpg"

	// This is a simple implementation - you may need to adjust based on your URL structure
	objectName := filepath.Base(imageURL)

	err := s.client.RemoveObject(ctx, s.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	log.Info().
		Str("object_name", objectName).
		Msg("Image deleted successfully")

	return nil
}

// GetImageURL returns the public URL for an image
func (s *MinIOStorage) GetImageURL(objectKey string) string {
	protocol := "http"
	if s.useSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", protocol, s.publicEndpoint, s.bucketName, objectKey)
}

// HealthCheck verifies the MinIO connection
func (s *MinIOStorage) HealthCheck(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("MinIO health check failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket '%s' does not exist", s.bucketName)
	}
	return nil
}
