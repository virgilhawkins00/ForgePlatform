package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Clear any existing env vars
	envVars := []string{
		"FORGE_DATA_DIR",
		"FORGE_LOG_LEVEL",
		"FORGE_HTTP_PORT",
		"FORGE_GCP_PROJECT_ID",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check defaults
	if cfg.Core.LogLevel != "info" {
		t.Errorf("Core.LogLevel = %v, want info", cfg.Core.LogLevel)
	}
	if cfg.Core.HTTPPort != 8080 {
		t.Errorf("Core.HTTPPort = %v, want 8080", cfg.Core.HTTPPort)
	}
	if cfg.Database.MaxConnections != 10 {
		t.Errorf("Database.MaxConnections = %v, want 10", cfg.Database.MaxConnections)
	}
	if cfg.GCP.Region != "southamerica-east1" {
		t.Errorf("GCP.Region = %v, want southamerica-east1", cfg.GCP.Region)
	}
	if cfg.GCP.BatchSize != 200 {
		t.Errorf("GCP.BatchSize = %v, want 200", cfg.GCP.BatchSize)
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	// Set env vars
	os.Setenv("FORGE_LOG_LEVEL", "debug")
	os.Setenv("FORGE_HTTP_PORT", "9090")
	os.Setenv("FORGE_GCP_PROJECT_ID", "test-project")
	defer func() {
		os.Unsetenv("FORGE_LOG_LEVEL")
		os.Unsetenv("FORGE_HTTP_PORT")
		os.Unsetenv("FORGE_GCP_PROJECT_ID")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Core.LogLevel != "debug" {
		t.Errorf("Core.LogLevel = %v, want debug", cfg.Core.LogLevel)
	}
	if cfg.GCP.ProjectID != "test-project" {
		t.Errorf("GCP.ProjectID = %v, want test-project", cfg.GCP.ProjectID)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Auth: AuthConfig{SessionTimeoutHours: 24},
			},
			wantErr: false,
		},
		{
			name: "invalid session timeout",
			config: Config{
				Auth: AuthConfig{SessionTimeoutHours: 0},
			},
			wantErr: true,
		},
		{
			name: "valid GCP config",
			config: Config{
				GCP: GCPConfig{
					ProjectID:     "test-project",
					BatchSize:     100,
					FlushInterval: 60 * time.Second,
				},
				Auth: AuthConfig{SessionTimeoutHours: 24},
			},
			wantErr: false,
		},
		{
			name: "invalid GCP batch size",
			config: Config{
				GCP: GCPConfig{
					ProjectID: "test-project",
					BatchSize: 0,
				},
				Auth: AuthConfig{SessionTimeoutHours: 24},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsGCPEnabled(t *testing.T) {
	cfg := &Config{}
	if cfg.IsGCPEnabled() {
		t.Error("IsGCPEnabled() = true, want false")
	}

	cfg.GCP.ProjectID = "test-project"
	if !cfg.IsGCPEnabled() {
		t.Error("IsGCPEnabled() = false, want true")
	}
}

func TestIsGCSEnabled(t *testing.T) {
	cfg := &Config{}
	if cfg.IsGCSEnabled() {
		t.Error("IsGCSEnabled() = true, want false")
	}

	cfg.GCS.Bucket = "test-bucket"
	if !cfg.IsGCSEnabled() {
		t.Error("IsGCSEnabled() = false, want true")
	}
}

