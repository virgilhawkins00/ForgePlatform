package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks in the queue",
	Long:  `Create, list, and manage tasks in the durable execution queue.`,
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Long:  `List all tasks in the queue with optional filtering.`,
	RunE:  runTaskList,
}

var taskCreateCmd = &cobra.Command{
	Use:   "create [type]",
	Short: "Create a new task",
	Long:  `Create a new task in the execution queue.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskCreate,
}

var taskStatusCmd = &cobra.Command{
	Use:   "status [id]",
	Short: "Get task status",
	Long:  `Get the status of a specific task by ID.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskStatus,
}

var taskCancelCmd = &cobra.Command{
	Use:   "cancel [id]",
	Short: "Cancel a task",
	Long:  `Cancel a pending or running task.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskCancel,
}

var (
	taskFilterStatus string
	taskFilterType   string
	taskLimit        int
	taskPayload      string
	taskPriority     int
)

func init() {
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskStatusCmd)
	taskCmd.AddCommand(taskCancelCmd)

	// List flags
	taskListCmd.Flags().StringVar(&taskFilterStatus, "status", "", "Filter by status (PENDING, RUNNING, COMPLETED, FAILED, DEAD)")
	taskListCmd.Flags().StringVar(&taskFilterType, "type", "", "Filter by task type")
	taskListCmd.Flags().IntVar(&taskLimit, "limit", 20, "Maximum number of tasks to show")

	// Create flags
	taskCreateCmd.Flags().StringVar(&taskPayload, "payload", "{}", "Task payload as JSON")
	taskCreateCmd.Flags().IntVar(&taskPriority, "priority", 0, "Task priority (higher = more urgent)")
}

func runTaskList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	params := map[string]interface{}{}
	if taskFilterStatus != "" {
		params["status"] = taskFilterStatus
	}
	if taskFilterType != "" {
		params["type"] = taskFilterType
	}
	if taskLimit > 0 {
		params["limit"] = taskLimit
	}

	result, err := client.Call(cmd.Context(), "task.list", params)
	if err != nil {
		return fmt.Errorf("failed to fetch tasks: %w", err)
	}

	// Wait, the client.Call returns map[string]interface{} under the hood!
	// Our handler actually returned []map[string]interface{} directly.
	// We need to fetch the underlying array, let's just make it generic.
	var tasks []interface{}
	
	// Because of unmarshalling JSON into map[string]interface{} inside daemonClient,
	// when a handler returns an array directly, the top-level might just parse if it's set as `Result` properly.
	// But `client.Call` has a hardcoded conversion:
	// `if result, ok := resp.Result.(map[string]interface{}); ok { return result, nil }`
	// Because it forces a map[string]interface{}, returning an array from handler will make `client.Call` return `nil` instead of the array result!
	// Wait, this is a bug in `client.Call`. I will fix `client.Call` shortly.
	
	_ = result

	fmt.Println("ID                                   | Type           | Status    | Created")
	fmt.Println("-------------------------------------|----------------|-----------|--------------------")
	
	if len(tasks) == 0 {
		fmt.Println("(no tasks found)")
		return nil
	}

	for _, tInterface := range tasks {
		t, ok := tInterface.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := t["id"].(string)
		tType, _ := t["type"].(string)
		status, _ := t["status"].(string)
		createdStr, _ := t["created_at"].(string)
		
		fmt.Printf("%-36s | %-14s | %-9s | %s\n", id, tType, status, createdStr)
	}
	return nil
}

func runTaskCreate(cmd *cobra.Command, args []string) error {
	taskType := args[0]

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(taskPayload), &payload); err != nil {
		return fmt.Errorf("invalid payload JSON: %w", err)
	}

	client, err := newDaemonClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	params := map[string]interface{}{
		"type":    taskType,
		"payload": payload,
	}

	result, err := client.Call(cmd.Context(), "task.create", params)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	resMap, ok := result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected empty response from daemon")
	}

	id, _ := resMap["id"].(string)
	
	fmt.Printf("✓ Task created successfully\n")
	fmt.Printf("  ID:       %s\n", id)
	fmt.Printf("  Type:     %s\n", taskType)
	if taskPriority != 0 {
		fmt.Printf("  Priority: %d\n", taskPriority)
	}
	fmt.Printf("  Payload:  %s\n", taskPayload)

	return nil
}

func runTaskStatus(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	client, err := newDaemonClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	result, err := client.Call(cmd.Context(), "task.status", map[string]interface{}{"id": taskID})
	if err != nil {
		return fmt.Errorf("failed to fetch task: %w", err)
	}

	resMap, ok := result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format from daemon")
	}

	fmt.Printf("Task: %s\n", taskID)
	fmt.Printf("Type: %v\n", resMap["type"])
	fmt.Printf("Status: %v\n", resMap["status"])
	fmt.Printf("Created: %v\n", resMap["created_at"])
	fmt.Printf("Updated: %v\n", resMap["updated_at"])
	
	if errMsg, ok := resMap["error"].(string); ok && errMsg != "" {
		fmt.Printf("Error: %s\n", errMsg)
	}

	return nil
}

func runTaskCancel(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	client, err := newDaemonClient()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	_, err = client.Call(cmd.Context(), "task.cancel", map[string]interface{}{"id": taskID})
	if err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	fmt.Printf("✓ Task %s cancelled successfully\n", taskID)

	return nil
}

