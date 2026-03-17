// Package cli implements the Cobra-based command-line interface for Forge.
package cli

import (
	"testing"
)

func TestVersionVariables(t *testing.T) {
	// Version variables should be defined with default values
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Commit == "" {
		t.Error("Commit should not be empty")
	}
	if BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}
}

func TestVersionVariables_DefaultValues(t *testing.T) {
	// Default values when not set via ldflags
	if Version != "dev" {
		t.Logf("Version is %s (expected 'dev' if not set via ldflags)", Version)
	}
	if Commit != "unknown" {
		t.Logf("Commit is %s (expected 'unknown' if not set via ldflags)", Commit)
	}
	if BuildDate != "unknown" {
		t.Logf("BuildDate is %s (expected 'unknown' if not set via ldflags)", BuildDate)
	}
}

func TestRootCmd_Defined(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}
	if rootCmd.Use != "forge" {
		t.Errorf("expected rootCmd.Use to be 'forge', got %s", rootCmd.Use)
	}
	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}
	if rootCmd.Long == "" {
		t.Error("rootCmd.Long should not be empty")
	}
}

func TestVersionCmd_Defined(t *testing.T) {
	if versionCmd == nil {
		t.Fatal("versionCmd should not be nil")
	}
	if versionCmd.Use != "version" {
		t.Errorf("expected versionCmd.Use to be 'version', got %s", versionCmd.Use)
	}
}

func TestHealthCmd_Defined(t *testing.T) {
	if healthCmd == nil {
		t.Fatal("healthCmd should not be nil")
	}
	if healthCmd.Use != "health" {
		t.Errorf("expected healthCmd.Use to be 'health', got %s", healthCmd.Use)
	}
}

func TestTaskCmd_Defined(t *testing.T) {
	if taskCmd == nil {
		t.Fatal("taskCmd should not be nil")
	}
	if taskCmd.Use != "task" {
		t.Errorf("expected taskCmd.Use to be 'task', got %s", taskCmd.Use)
	}
}

func TestMetricCmd_Defined(t *testing.T) {
	if metricCmd == nil {
		t.Fatal("metricCmd should not be nil")
	}
	if metricCmd.Use != "metric" {
		t.Errorf("expected metricCmd.Use to be 'metric', got %s", metricCmd.Use)
	}
}

func TestPluginCmd_Defined(t *testing.T) {
	if pluginCmd == nil {
		t.Fatal("pluginCmd should not be nil")
	}
	if pluginCmd.Use != "plugin" {
		t.Errorf("expected pluginCmd.Use to be 'plugin', got %s", pluginCmd.Use)
	}
}

func TestWorkflowCmd_Defined(t *testing.T) {
	if workflowCmd == nil {
		t.Fatal("workflowCmd should not be nil")
	}
	if workflowCmd.Use != "workflow" {
		t.Errorf("expected workflowCmd.Use to be 'workflow', got %s", workflowCmd.Use)
	}
}

func TestAlertCmd_Defined(t *testing.T) {
	if alertCmd == nil {
		t.Fatal("alertCmd should not be nil")
	}
	if alertCmd.Use != "alert" {
		t.Errorf("expected alertCmd.Use to be 'alert', got %s", alertCmd.Use)
	}
}

func TestAICmd_Defined(t *testing.T) {
	if aiCmd == nil {
		t.Fatal("aiCmd should not be nil")
	}
	if aiCmd.Use != "ai" {
		t.Errorf("expected aiCmd.Use to be 'ai', got %s", aiCmd.Use)
	}
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	subcommands := rootCmd.Commands()
	if len(subcommands) == 0 {
		t.Error("rootCmd should have subcommands")
	}

	// Check that expected subcommands are registered
	expectedCommands := []string{"version", "health", "task", "metric", "plugin", "workflow", "alert", "ai"}
	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range subcommands {
			if cmd.Use == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand '%s' not found", expected)
		}
	}
}

