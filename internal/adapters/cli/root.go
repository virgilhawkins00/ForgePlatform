// Package cli implements the Cobra-based command-line interface for Forge.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	v       *viper.Viper
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "forge",
	Short: "Forge - Unified Engineering Platform",
	Long: `Forge is a unified engineering platform that combines:
  • CLI - Command-line interface for task automation
  • TUI - Terminal user interface for interactive dashboards
  • TSDB - Embedded time-series database for metrics
  • Wasm - WebAssembly plugin system for extensibility
  • AI - Local LLM integration for intelligent assistance

All components are bundled into a single binary for maximum portability.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd)
	},
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.forge/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(taskCmd)
	rootCmd.AddCommand(metricCmd)
	rootCmd.AddCommand(pluginCmd)
	rootCmd.AddCommand(aiCmd)
	rootCmd.AddCommand(uiCmd)
	rootCmd.AddCommand(workflowCmd)
}

// initializeConfig reads in config file and ENV variables if set.
func initializeConfig(cmd *cobra.Command) error {
	v = viper.New()

	// Set config file
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		forgeDir := filepath.Join(home, ".forge")
		v.AddConfigPath(forgeDir)
		v.AddConfigPath(".")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// Environment variables
	v.SetEnvPrefix("FORGE")
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}

	// Bind flags to viper
	if err := bindFlags(cmd, v); err != nil {
		return err
	}

	return nil
}

// bindFlags binds command flags to viper configuration.
func bindFlags(cmd *cobra.Command, v *viper.Viper) error {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Bind the flag to viper
		configName := f.Name
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			_ = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
	return nil
}

// getForgeDir returns the Forge configuration directory.
func getForgeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".forge"), nil
}

// ensureForgeDir creates the Forge directory if it doesn't exist.
func ensureForgeDir() (string, error) {
	dir, err := getForgeDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

