// Package cloud provides cloud provider integrations.
package cloud

import (
	"testing"
)

func TestDefaultGCSConfig(t *testing.T) {
	cfg := DefaultGCSConfig()

	if cfg.BackupPrefix != "backups/" {
		t.Errorf("expected BackupPrefix 'backups/', got '%s'", cfg.BackupPrefix)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("expected RetentionDays 30, got %d", cfg.RetentionDays)
	}
	if cfg.Bucket != "" {
		t.Errorf("expected empty Bucket, got '%s'", cfg.Bucket)
	}
	if cfg.CredentialsPath != "" {
		t.Errorf("expected empty CredentialsPath, got '%s'", cfg.CredentialsPath)
	}
}

