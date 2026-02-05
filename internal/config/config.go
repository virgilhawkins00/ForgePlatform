// Package config provides typed configuration management using Viper.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Core     CoreConfig     `mapstructure:"core"`
	Database DatabaseConfig `mapstructure:"database"`
	GCP      GCPConfig      `mapstructure:"gcp"`
	GCS      GCSConfig      `mapstructure:"gcs"`
	Auth     AuthConfig     `mapstructure:"auth"`
	AI       AIConfig       `mapstructure:"ai"`
	Alerting AlertingConfig `mapstructure:"alerting"`
	Dev      DevConfig      `mapstructure:"dev"`
}

// CoreConfig holds core application settings.
type CoreConfig struct {
	DataDir  string `mapstructure:"data_dir"`
	LogLevel string `mapstructure:"log_level"`
	HTTPPort int    `mapstructure:"http_port"`
}

// DatabaseConfig holds database settings.
type DatabaseConfig struct {
	Path           string `mapstructure:"path"`
	MaxConnections int    `mapstructure:"max_connections"`
	CacheSize      int    `mapstructure:"cache_size"`
}

// GCPConfig holds GCP Cloud Monitoring settings.
type GCPConfig struct {
	ProjectID       string        `mapstructure:"project_id"`
	CredentialsPath string        `mapstructure:"credentials_path"`
	Region          string        `mapstructure:"region"`
	MetricPrefix    string        `mapstructure:"metric_prefix"`
	FlushInterval   time.Duration `mapstructure:"flush_interval"`
	BatchSize       int           `mapstructure:"batch_size"`
}

// GCSConfig holds Google Cloud Storage settings.
type GCSConfig struct {
	Bucket            string `mapstructure:"bucket"`
	BackupRetentionDays int  `mapstructure:"backup_retention_days"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTSecret           string `mapstructure:"jwt_secret"`
	SessionTimeoutHours int    `mapstructure:"session_timeout_hours"`
	APIKeySalt          string `mapstructure:"api_key_salt"`
}

// AIConfig holds AI/LLM settings.
type AIConfig struct {
	OllamaURL string `mapstructure:"ollama_url"`
	Model     string `mapstructure:"model"`
}

// AlertingConfig holds alerting settings.
type AlertingConfig struct {
	SlackWebhookURL string     `mapstructure:"slack_webhook_url"`
	PagerDutyKey    string     `mapstructure:"pagerduty_key"`
	SMTP            SMTPConfig `mapstructure:"smtp"`
}

// SMTPConfig holds SMTP settings.
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

// DevConfig holds development settings.
type DevConfig struct {
	Debug            bool `mapstructure:"debug"`
	ProfilingEnabled bool `mapstructure:"profiling_enabled"`
}

// Load loads configuration from environment and config files.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Load .env file if exists (for local development)
	if err := loadEnvFile(v); err != nil {
		// .env file is optional, don't fail
		_ = err
	}

	// Environment variables with FORGE_ prefix
	v.SetEnvPrefix("FORGE")
	v.AutomaticEnv()

	// Bind environment variables to config keys
	bindEnvVars(v)

	// Load config file if exists
	if err := loadConfigFile(v); err != nil {
		// Config file is optional
		_ = err
	}

	// Unmarshal into struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// Core defaults
	v.SetDefault("core.data_dir", getDefaultDataDir())
	v.SetDefault("core.log_level", "info")
	v.SetDefault("core.http_port", 8080)

	// Database defaults
	v.SetDefault("database.max_connections", 10)
	v.SetDefault("database.cache_size", 64000)

	// GCP defaults
	v.SetDefault("gcp.region", "southamerica-east1")
	v.SetDefault("gcp.metric_prefix", "custom.googleapis.com/forge")
	v.SetDefault("gcp.flush_interval", 60*time.Second)
	v.SetDefault("gcp.batch_size", 200)

	// GCS defaults
	v.SetDefault("gcs.backup_retention_days", 30)

	// Auth defaults
	v.SetDefault("auth.session_timeout_hours", 24)

	// AI defaults
	v.SetDefault("ai.ollama_url", "http://localhost:11434")
	v.SetDefault("ai.model", "llama3.2")

	// Alerting defaults
	v.SetDefault("alerting.smtp.port", 587)

	// Dev defaults
	v.SetDefault("dev.debug", false)
	v.SetDefault("dev.profiling_enabled", false)
}

// bindEnvVars binds environment variables to config keys.
func bindEnvVars(v *viper.Viper) {
	// Core
	_ = v.BindEnv("core.data_dir", "FORGE_DATA_DIR")
	_ = v.BindEnv("core.log_level", "FORGE_LOG_LEVEL")
	_ = v.BindEnv("core.http_port", "FORGE_HTTP_PORT")

	// Database
	_ = v.BindEnv("database.path", "FORGE_DB_PATH")
	_ = v.BindEnv("database.max_connections", "FORGE_DB_MAX_CONNECTIONS")
	_ = v.BindEnv("database.cache_size", "FORGE_DB_CACHE_SIZE")

	// GCP
	_ = v.BindEnv("gcp.project_id", "FORGE_GCP_PROJECT_ID")
	_ = v.BindEnv("gcp.credentials_path", "FORGE_GCP_CREDENTIALS_PATH")
	_ = v.BindEnv("gcp.region", "FORGE_GCP_REGION")
	_ = v.BindEnv("gcp.metric_prefix", "FORGE_GCP_METRIC_PREFIX")
	_ = v.BindEnv("gcp.flush_interval", "FORGE_GCP_FLUSH_INTERVAL")
	_ = v.BindEnv("gcp.batch_size", "FORGE_GCP_BATCH_SIZE")

	// GCS
	_ = v.BindEnv("gcs.bucket", "FORGE_GCS_BUCKET")
	_ = v.BindEnv("gcs.backup_retention_days", "FORGE_BACKUP_RETENTION_DAYS")

	// Auth
	_ = v.BindEnv("auth.jwt_secret", "FORGE_JWT_SECRET")
	_ = v.BindEnv("auth.session_timeout_hours", "FORGE_SESSION_TIMEOUT_HOURS")
	_ = v.BindEnv("auth.api_key_salt", "FORGE_API_KEY_SALT")

	// AI
	_ = v.BindEnv("ai.ollama_url", "FORGE_OLLAMA_URL")
	_ = v.BindEnv("ai.model", "FORGE_AI_MODEL")

	// Alerting
	_ = v.BindEnv("alerting.slack_webhook_url", "FORGE_SLACK_WEBHOOK_URL")
	_ = v.BindEnv("alerting.pagerduty_key", "FORGE_PAGERDUTY_KEY")
	_ = v.BindEnv("alerting.smtp.host", "FORGE_SMTP_HOST")
	_ = v.BindEnv("alerting.smtp.port", "FORGE_SMTP_PORT")
	_ = v.BindEnv("alerting.smtp.username", "FORGE_SMTP_USERNAME")
	_ = v.BindEnv("alerting.smtp.password", "FORGE_SMTP_PASSWORD")
	_ = v.BindEnv("alerting.smtp.from", "FORGE_SMTP_FROM")

	// Dev
	_ = v.BindEnv("dev.debug", "FORGE_DEBUG")
	_ = v.BindEnv("dev.profiling_enabled", "FORGE_PROFILING_ENABLED")
}

// loadEnvFile loads .env file if it exists.
func loadEnvFile(v *viper.Viper) error {
	// Check for .env in current directory
	if _, err := os.Stat(".env"); err == nil {
		v.SetConfigFile(".env")
		v.SetConfigType("env")
		return v.MergeInConfig()
	}
	return nil
}

// loadConfigFile loads config.yaml if it exists.
func loadConfigFile(v *viper.Viper) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	forgeDir := filepath.Join(home, ".forge")
	v.AddConfigPath(forgeDir)
	v.AddConfigPath(".")
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	return v.MergeInConfig()
}

// getDefaultDataDir returns the default data directory.
func getDefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".forge/data"
	}
	return filepath.Join(home, ".forge", "data")
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// GCP validation
	if c.GCP.ProjectID != "" {
		if c.GCP.BatchSize <= 0 {
			return fmt.Errorf("gcp.batch_size must be positive")
		}
		if c.GCP.FlushInterval <= 0 {
			return fmt.Errorf("gcp.flush_interval must be positive")
		}
	}

	// Auth validation
	if c.Auth.SessionTimeoutHours <= 0 {
		return fmt.Errorf("auth.session_timeout_hours must be positive")
	}

	return nil
}

// IsGCPEnabled returns true if GCP integration is configured.
func (c *Config) IsGCPEnabled() bool {
	return c.GCP.ProjectID != ""
}

// IsGCSEnabled returns true if GCS backup is configured.
func (c *Config) IsGCSEnabled() bool {
	return c.GCS.Bucket != ""
}

