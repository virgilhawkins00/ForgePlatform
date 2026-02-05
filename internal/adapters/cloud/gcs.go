// Package cloud provides cloud provider integrations.
package cloud

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/forge-platform/forge/internal/core/ports"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GCSConfig holds Google Cloud Storage configuration.
type GCSConfig struct {
	Bucket            string `json:"bucket"`
	CredentialsPath   string `json:"credentials_path,omitempty"`
	BackupPrefix      string `json:"backup_prefix"`
	RetentionDays     int    `json:"retention_days"`
}

// DefaultGCSConfig returns default GCS configuration.
func DefaultGCSConfig() GCSConfig {
	return GCSConfig{
		BackupPrefix:  "backups/",
		RetentionDays: 30,
	}
}

// GCSBackupService provides backup/restore operations using GCS.
type GCSBackupService struct {
	config GCSConfig
	client *storage.Client
	logger ports.Logger
}

// NewGCSBackupService creates a new GCS backup service.
func NewGCSBackupService(ctx context.Context, config GCSConfig, logger ports.Logger) (*GCSBackupService, error) {
	if config.Bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	var opts []option.ClientOption
	if config.CredentialsPath != "" {
		if _, err := os.Stat(config.CredentialsPath); err != nil {
			return nil, fmt.Errorf("credentials file not found: %s", config.CredentialsPath)
		}
		opts = append(opts, option.WithCredentialsFile(config.CredentialsPath))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &GCSBackupService{
		config: config,
		client: client,
		logger: logger,
	}, nil
}

// BackupInfo represents information about a backup.
type BackupInfo struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Path      string    `json:"path"`
}

// Upload uploads a backup file to GCS.
func (s *GCSBackupService) Upload(ctx context.Context, localPath string) (*BackupInfo, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Generate object name with timestamp
	objectName := fmt.Sprintf("%s%s", s.config.BackupPrefix, filepath.Base(localPath))

	bucket := s.client.Bucket(s.config.Bucket)
	obj := bucket.Object(objectName)
	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/gzip"

	if _, err := io.Copy(writer, file); err != nil {
		return nil, fmt.Errorf("failed to upload: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	s.logger.Info("Backup uploaded to GCS", "bucket", s.config.Bucket, "object", objectName)

	return &BackupInfo{
		Name:      filepath.Base(localPath),
		Size:      stat.Size(),
		CreatedAt: time.Now(),
		Path:      fmt.Sprintf("gs://%s/%s", s.config.Bucket, objectName),
	}, nil
}

// Download downloads a backup from GCS.
func (s *GCSBackupService) Download(ctx context.Context, objectName, localPath string) error {
	bucket := s.client.Bucket(s.config.Bucket)
	obj := bucket.Object(s.config.BackupPrefix + objectName)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	s.logger.Info("Backup downloaded from GCS", "object", objectName, "path", localPath)
	return nil
}

// List lists all backups in GCS.
func (s *GCSBackupService) List(ctx context.Context) ([]BackupInfo, error) {
	bucket := s.client.Bucket(s.config.Bucket)
	query := &storage.Query{Prefix: s.config.BackupPrefix}

	var backups []BackupInfo
	it := bucket.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		name := strings.TrimPrefix(attrs.Name, s.config.BackupPrefix)
		backups = append(backups, BackupInfo{
			Name:      name,
			Size:      attrs.Size,
			CreatedAt: attrs.Created,
			Path:      fmt.Sprintf("gs://%s/%s", s.config.Bucket, attrs.Name),
		})
	}

	return backups, nil
}

// Delete deletes a backup from GCS.
func (s *GCSBackupService) Delete(ctx context.Context, objectName string) error {
	bucket := s.client.Bucket(s.config.Bucket)
	obj := bucket.Object(s.config.BackupPrefix + objectName)

	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	s.logger.Info("Backup deleted from GCS", "object", objectName)
	return nil
}

// Cleanup removes backups older than retention period.
func (s *GCSBackupService) Cleanup(ctx context.Context) (int, error) {
	if s.config.RetentionDays <= 0 {
		return 0, nil
	}

	cutoff := time.Now().AddDate(0, 0, -s.config.RetentionDays)
	backups, err := s.List(ctx)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, backup := range backups {
		if backup.CreatedAt.Before(cutoff) {
			if err := s.Delete(ctx, backup.Name); err != nil {
				s.logger.Error("Failed to delete old backup", "name", backup.Name, "error", err)
				continue
			}
			deleted++
		}
	}

	s.logger.Info("Cleanup completed", "deleted", deleted, "retention_days", s.config.RetentionDays)
	return deleted, nil
}

// Close closes the GCS client.
func (s *GCSBackupService) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// GetConfig returns the current configuration.
func (s *GCSBackupService) GetConfig() GCSConfig {
	return s.config
}

