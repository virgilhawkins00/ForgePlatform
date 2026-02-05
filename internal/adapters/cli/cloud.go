// Package cli provides CLI commands for cloud integration.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var cloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Cloud provider integration commands",
	Long:  `Manage cloud provider integrations for metrics export and monitoring.`,
}

var cloudGCPCmd = &cobra.Command{
	Use:   "gcp",
	Short: "GCP Cloud Monitoring integration",
	Long:  `Configure and manage GCP Cloud Monitoring integration.`,
}

var cloudGCPConfigureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure GCP Cloud Monitoring",
	Long:  `Configure GCP Cloud Monitoring integration with project ID and credentials.`,
	RunE:  runGCPConfigure,
}

var cloudGCPStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show GCP integration status",
	Long:  `Display the current GCP Cloud Monitoring integration status.`,
	RunE:  runGCPStatus,
}

// GCP configuration flags
var (
	gcpProjectID       string
	gcpCredentialsPath string
	gcpRegion          string
	gcpMetricPrefix    string
)

func init() {
	cloudGCPConfigureCmd.Flags().StringVar(&gcpProjectID, "project-id", "", "GCP project ID (required)")
	cloudGCPConfigureCmd.Flags().StringVar(&gcpCredentialsPath, "credentials", "", "Path to service account credentials JSON")
	cloudGCPConfigureCmd.Flags().StringVar(&gcpRegion, "region", "us-central1", "GCP region")
	cloudGCPConfigureCmd.Flags().StringVar(&gcpMetricPrefix, "metric-prefix", "custom.googleapis.com/forge", "Metric prefix for Cloud Monitoring")
	cloudGCPConfigureCmd.MarkFlagRequired("project-id")

	cloudGCPCmd.AddCommand(cloudGCPConfigureCmd)
	cloudGCPCmd.AddCommand(cloudGCPStatusCmd)
	cloudCmd.AddCommand(cloudGCPCmd)
}

// GCPCloudConfig represents the GCP configuration file.
type GCPCloudConfig struct {
	ProjectID       string `json:"project_id"`
	CredentialsPath string `json:"credentials_path,omitempty"`
	Region          string `json:"region"`
	MetricPrefix    string `json:"metric_prefix"`
	Enabled         bool   `json:"enabled"`
}

func runGCPConfigure(cmd *cobra.Command, args []string) error {
	config := GCPCloudConfig{
		ProjectID:       gcpProjectID,
		CredentialsPath: gcpCredentialsPath,
		Region:          gcpRegion,
		MetricPrefix:    gcpMetricPrefix,
		Enabled:         true,
	}

	// Create config directory if it doesn't exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".forge", "cloud")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write configuration
	configPath := filepath.Join(configDir, "gcp.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✅ GCP Cloud Monitoring configured successfully!\n\n")
	fmt.Printf("Configuration saved to: %s\n", configPath)
	fmt.Printf("Project ID:    %s\n", config.ProjectID)
	fmt.Printf("Region:        %s\n", config.Region)
	fmt.Printf("Metric Prefix: %s\n", config.MetricPrefix)

	if config.CredentialsPath != "" {
		fmt.Printf("Credentials:   %s\n", config.CredentialsPath)
	} else {
		fmt.Printf("Credentials:   Using default (ADC/Workload Identity)\n")
	}

	return nil
}

func runGCPStatus(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".forge", "cloud", "gcp.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("❌ GCP Cloud Monitoring not configured")
			fmt.Println("\nRun 'forge cloud gcp configure --project-id <PROJECT_ID>' to set up")
			return nil
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config GCPCloudConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	status := "✅ Enabled"
	if !config.Enabled {
		status = "❌ Disabled"
	}

	fmt.Printf("GCP Cloud Monitoring Status\n")
	fmt.Printf("===========================\n\n")
	fmt.Printf("Status:        %s\n", status)
	fmt.Printf("Project ID:    %s\n", config.ProjectID)
	fmt.Printf("Region:        %s\n", config.Region)
	fmt.Printf("Metric Prefix: %s\n", config.MetricPrefix)

	return nil
}

