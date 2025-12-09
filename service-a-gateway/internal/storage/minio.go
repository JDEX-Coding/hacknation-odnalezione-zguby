package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
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

	// Clean public endpoint: strip trailing slash and whitespace/quotes
	publicEndpoint = strings.TrimSpace(publicEndpoint)
	publicEndpoint = strings.Trim(publicEndpoint, `"'=`)
	publicEndpoint = strings.TrimSuffix(publicEndpoint, "/")

	storage := &MinIOStorage{
		client:         minioClient,
		bucketName:     bucketName,
		endpoint:       endpoint,
		publicEndpoint: publicEndpoint,
		useSSL:         useSSL,
	}

	log.Info().
		Str("internal_endpoint", endpoint).
		Str("public_endpoint", publicEndpoint).
		Str("bucket", bucketName).
		Bool("use_ssl", useSSL).
		Msg("MinIO storage configuration initialized")

	// Verify bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Only verify bucket if we can reach the endpoint (skip if external/unreachable from container)
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to check bucket existence for %s (will continue)", bucketName)
	} else if !exists {
		// Try to create bucket if it doesn't exist
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create bucket %s", bucketName)
		} else {
			log.Info().Msgf("Bucket %s created successfully", bucketName)
		}
	}

	// Always ensure policy is public read (fixes existing buckets with wrong policy)
	policy := fmt.Sprintf(`{"Version": "2012-10-17","Statement": [{"Action": ["s3:GetObject"],"Effect": "Allow","Principal": {"AWS": ["*"]},"Resource": ["arn:aws:s3:::%s/*"],"Sid": ""}]}`, bucketName)
	if err := minioClient.SetBucketPolicy(ctx, bucketName, policy); err != nil {
		log.Error().Err(err).Msg("Failed to set bucket policy")
	} else {
		log.Info().Msg("Verified/Set public bucket policy")
	}

	log.Info().
		Str("endpoint", endpoint).
		Str("public_endpoint", publicEndpoint).
		Str("bucket", bucketName).
		Msg("MinIO storage initialized")

	return storage, nil
}

// UploadImage uploads an image to MinIO and returns the object key and public URL
func (s *MinIOStorage) UploadImage(ctx context.Context, reader io.Reader, filename string, contentType string, size int64) (string, string, error) {
	// Generate unique filename/key
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
		return "", "", fmt.Errorf("failed to upload image: %w", err)
	}

	publicURL := s.GetImageURL(uniqueFilename)

	log.Info().
		Str("filename", filename).
		Str("key", uniqueFilename).
		Str("url", publicURL).
		Msg("Image uploaded successfully")

	return uniqueFilename, publicURL, nil
}

// DeleteImage deletes an image from MinIO
func (s *MinIOStorage) DeleteImage(ctx context.Context, imageURL string) error {
	// Extract object key from URL
	objectName := s.GetKeyFromURL(imageURL)
	if objectName == "" {
		return fmt.Errorf("could not extract key from URL")
	}

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
	// Use direct public URL with path-style access
	// Format: https://minio.jdex.pl/bucket-name/object-key
	// This works because:
	// 1. Bucket policy is set to public read in NewMinIOStorage
	// 2. Traefik routes minio.jdex.pl to MinIO on port 9000
	// 3. MinIO serves files directly at /bucket-name/object-key

	cleanEndpoint := strings.Trim(s.publicEndpoint, "\"'= ")

	// Ensure we have the scheme
	if !strings.Contains(cleanEndpoint, "://") {
		protocol := "http"
		if s.useSSL {
			protocol = "https"
		}
		cleanEndpoint = fmt.Sprintf("%s://%s", protocol, cleanEndpoint)
	}

	// Build the direct URL: https://minio.jdex.pl/bucket-name/object-key
	finalURL := fmt.Sprintf("%s/%s/%s", cleanEndpoint, s.bucketName, objectKey)

	log.Info().
		Str("object_key", objectKey).
		Str("bucket", s.bucketName).
		Str("public_endpoint", cleanEndpoint).
		Str("final_url", finalURL).
		Msg("Generated public URL for image")

	return finalURL
}

// GetKeyFromURL attempts to extract the object key from a full URL
func (s *MinIOStorage) GetKeyFromURL(imageURL string) string {
	// Try parsing URL
	u, err := url.Parse(imageURL)
	if err != nil {
		return ""
	}

	// Path should look like /bucketName/path/to/object
	path := strings.TrimPrefix(u.Path, "/")

	// Check if path starts with bucket name
	prefix := s.bucketName + "/"
	if strings.HasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix)
	}

	// Fallback: search for bucketName in the path (handles malformed URLs)
	// Example: //minio.jdex.com/lost-items-images//minio.jdex.com/lost-items-images/uploads/...
	// Use LastIndex to find the actual key if the bucket name is duplicated
	if idx := strings.LastIndex(path, prefix); idx != -1 {
		return path[idx+len(prefix):]
	}

	return path
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
