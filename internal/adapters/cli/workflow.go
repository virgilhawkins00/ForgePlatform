package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage and execute workflows",
	Long:  `Define, execute, and monitor YAML-based automation workflows.`,
}

var workflowRunCmd = &cobra.Command{
	Use:   "run [file.yaml]",
	Short: "Run a workflow from a YAML file",
	Long:  `Load and execute a workflow definition from a YAML file.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowRun,
}

var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflow definitions",
	Long:  `List all registered workflow definitions.`,
	RunE:  runWorkflowList,
}

var workflowStatusCmd = &cobra.Command{
	Use:   "status [execution-id]",
	Short: "Get workflow execution status",
	Long:  `Get the current status of a workflow execution.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowStatus,
}

var workflowCancelCmd = &cobra.Command{
	Use:   "cancel [execution-id]",
	Short: "Cancel a running workflow",
	Long:  `Cancel a running workflow execution.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowCancel,
}

var workflowHistoryCmd = &cobra.Command{
	Use:   "history [workflow-name]",
	Short: "Show workflow execution history",
	Long:  `Show the execution history for a specific workflow.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWorkflowHistory,
}

var (
	workflowInput   string
	workflowAsync   bool
	workflowVerbose bool
	historyLimit    int
)

func init() {
	workflowCmd.AddCommand(workflowRunCmd)
	workflowCmd.AddCommand(workflowListCmd)
	workflowCmd.AddCommand(workflowStatusCmd)
	workflowCmd.AddCommand(workflowCancelCmd)
	workflowCmd.AddCommand(workflowHistoryCmd)

	// Run flags
	workflowRunCmd.Flags().StringVarP(&workflowInput, "input", "i", "{}", "Input variables as JSON")
	workflowRunCmd.Flags().BoolVarP(&workflowAsync, "async", "a", false, "Run workflow asynchronously")
	workflowRunCmd.Flags().BoolVarP(&workflowVerbose, "verbose", "v", false, "Show verbose output")

	// History flags
	workflowHistoryCmd.Flags().IntVarP(&historyLimit, "limit", "n", 10, "Maximum number of executions to show")
}

func runWorkflowRun(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("workflow file not found: %s", filePath)
	}

	// Parse input
	var input map[string]interface{}
	if err := json.Unmarshal([]byte(workflowInput), &input); err != nil {
		return fmt.Errorf("invalid input JSON: %w", err)
	}

	// Connect to daemon
	client, err := newDaemonClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Call daemon to run workflow
	resp, err := client.Call(ctx, "workflow.run", map[string]interface{}{
		"file":  filePath,
		"input": input,
		"async": workflowAsync,
	})
	if err != nil {
		return fmt.Errorf("failed to run workflow: %w", err)
	}

	if workflowAsync {
		fmt.Printf("✅ Workflow started (execution ID: %s)\n", resp["execution_id"])
		fmt.Println("Use 'forge workflow status <id>' to check progress")
	} else {
		printWorkflowResult(resp)
	}

	return nil
}

func runWorkflowList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "workflow.list", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	workflows, ok := resp["workflows"].([]interface{})
	if !ok || len(workflows) == 0 {
		fmt.Println("No workflow definitions found.")
		return nil
	}

	fmt.Println("NAME                     | VERSION | STEPS | DESCRIPTION")
	fmt.Println("-------------------------|---------|-------|---------------------------")
	for _, w := range workflows {
		wf := w.(map[string]interface{})
		name := truncate(getString(wf, "name"), 24)
		version := getString(wf, "version")
		steps := len(wf["steps"].([]interface{}))
		desc := truncate(getString(wf, "description"), 26)
		fmt.Printf("%-24s | %-7s | %-5d | %s\n", name, version, steps, desc)
	}

	return nil
}

func runWorkflowStatus(cmd *cobra.Command, args []string) error {
	executionID := args[0]

	client, err := newDaemonClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "workflow.status", map[string]interface{}{
		"execution_id": executionID,
	})
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	printExecutionStatus(resp)
	return nil
}

func runWorkflowCancel(cmd *cobra.Command, args []string) error {
	executionID := args[0]

	client, err := newDaemonClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	ctx := context.Background()
	_, err = client.Call(ctx, "workflow.cancel", map[string]interface{}{
		"execution_id": executionID,
	})
	if err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}

	fmt.Printf("✅ Workflow execution %s cancelled\n", executionID)
	return nil
}

func runWorkflowHistory(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	params := map[string]interface{}{
		"limit": historyLimit,
	}
	if len(args) > 0 {
		params["workflow_name"] = args[0]
	}

	ctx := context.Background()
	resp, err := client.Call(ctx, "workflow.history", params)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	executions, ok := resp["executions"].([]interface{})
	if !ok || len(executions) == 0 {
		fmt.Println("No execution history found.")
		return nil
	}

	fmt.Println("EXECUTION ID                         | WORKFLOW         | STATUS    | STARTED              | DURATION")
	fmt.Println("-------------------------------------|------------------|-----------|----------------------|---------")
	for _, e := range executions {
		exec := e.(map[string]interface{})
		id := truncate(getString(exec, "id"), 36)
		name := truncate(getString(exec, "workflow_name"), 16)
		status := statusIcon(getString(exec, "status"))
		started := formatTime(exec["started_at"])
		duration := formatDuration(exec["duration"])
		fmt.Printf("%-36s | %-16s | %-9s | %-20s | %s\n", id, name, status, started, duration)
	}

	return nil
}

// Helper functions

func printWorkflowResult(resp map[string]interface{}) {
	status := getString(resp, "status")
	fmt.Printf("\n📋 Workflow: %s\n", getString(resp, "workflow_name"))
	fmt.Printf("   Status: %s\n", statusIcon(status))

	if duration, ok := resp["duration"]; ok {
		fmt.Printf("   Duration: %s\n", formatDuration(duration))
	}

	if steps, ok := resp["steps"].([]interface{}); ok {
		fmt.Println("\n   Steps:")
		for _, s := range steps {
			step := s.(map[string]interface{})
			stepStatus := statusIcon(getString(step, "status"))
			fmt.Printf("   %s %s (%s)\n", stepStatus, getString(step, "step_name"), formatDuration(step["duration"]))
		}
	}

	if errMsg := getString(resp, "error"); errMsg != "" {
		fmt.Printf("\n   ❌ Error: %s\n", errMsg)
	}

	fmt.Println()
}

func printExecutionStatus(resp map[string]interface{}) {
	fmt.Printf("\n📋 Execution: %s\n", getString(resp, "id"))
	fmt.Printf("   Workflow: %s\n", getString(resp, "workflow_name"))
	fmt.Printf("   Status: %s\n", statusIcon(getString(resp, "status")))
	fmt.Printf("   Started: %s\n", formatTime(resp["started_at"]))

	if steps, ok := resp["steps"].([]interface{}); ok {
		fmt.Println("\n   Steps:")
		for _, s := range steps {
			step := s.(map[string]interface{})
			stepStatus := statusIcon(getString(step, "status"))
			retries := 0
			if r, ok := step["retry_count"].(float64); ok {
				retries = int(r)
			}
			retryStr := ""
			if retries > 0 {
				retryStr = fmt.Sprintf(" (retries: %d)", retries)
			}
			fmt.Printf("   %s %s%s\n", stepStatus, getString(step, "step_name"), retryStr)
		}
	}

	if errMsg := getString(resp, "error"); errMsg != "" {
		fmt.Printf("\n   ❌ Error: %s\n", errMsg)
	}

	fmt.Println()
}

func statusIcon(status string) string {
	switch strings.ToLower(status) {
	case "completed":
		return "✅ completed"
	case "running":
		return "🔄 running"
	case "pending":
		return "⏳ pending"
	case "failed":
		return "❌ failed"
	case "cancelled":
		return "🚫 cancelled"
	default:
		return status
	}
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatTime(v interface{}) string {
	if s, ok := v.(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
		return s
	}
	return "-"
}

func formatDuration(v interface{}) string {
	switch d := v.(type) {
	case float64:
		return time.Duration(int64(d)).String()
	case int64:
		return time.Duration(d).String()
	case string:
		return d
	default:
		return "-"
	}
}

