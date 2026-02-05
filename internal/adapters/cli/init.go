package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Forge configuration",
	Long: `Initialize the Forge platform by creating the configuration directory
and default configuration files.

This command creates:
  • ~/.forge/config.yaml - Main configuration file
  • ~/.forge/plugins/ - Plugin directory
  • ~/.forge/data/ - Data directory for SQLite databases
  • ~/.forge/logs/ - Log files directory`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	forgeDir, err := ensureForgeDir()
	if err != nil {
		return fmt.Errorf("failed to create forge directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"plugins", "data", "logs"}
	for _, subdir := range subdirs {
		path := filepath.Join(forgeDir, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
		fmt.Printf("✓ Created %s\n", path)
	}

	// Create default config file
	configPath := filepath.Join(forgeDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
		fmt.Printf("✓ Created %s\n", configPath)
	} else {
		fmt.Printf("• Config file already exists: %s\n", configPath)
	}

	fmt.Println("\n🔧 Forge initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Edit ~/.forge/config.yaml to customize settings")
	fmt.Println("  2. Run 'forge start' to start the daemon")
	fmt.Println("  3. Run 'forge ui' to open the terminal interface")

	return nil
}

const defaultConfig = `# Forge Platform Configuration
# https://github.com/forge-platform/forge

# General settings
general:
  log_level: info  # debug, info, warn, error
  data_dir: ~/.forge/data
  plugin_dir: ~/.forge/plugins

# Database settings (SQLite TSDB)
database:
  path: ~/.forge/data/forge.db
  # SQLite optimizations for TSDB
  pragmas:
    journal_mode: WAL
    synchronous: NORMAL
    cache_size: -64000  # 64MB
    mmap_size: 268435456  # 256MB
    busy_timeout: 5000

# Metrics retention policy
metrics:
  raw_retention: 7d      # Keep raw data for 7 days
  medium_retention: 30d  # Keep 1-minute aggregates for 30 days
  long_retention: 365d   # Keep 1-hour aggregates for 1 year
  downsample_interval: 1h  # Run downsampling every hour

# Daemon settings
daemon:
  socket_path: ~/.forge/forge.sock
  pid_file: ~/.forge/forge.pid
  shutdown_timeout: 10s
  worker_count: 4

# AI settings
ai:
  provider: ollama
  model: llama3.2
  endpoint: http://localhost:11434
  context_window: 8192
  temperature: 0.7

# Plugin settings
plugins:
  auto_load: true
  sandbox:
    memory_limit: 256MB
    timeout: 30s
    allowed_hosts: []

# TUI settings
tui:
  refresh_rate: 1s
  theme: dark
  show_graphs: true
`

