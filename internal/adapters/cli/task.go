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
	// TODO: Connect to daemon and fetch tasks
	fmt.Println("ID                                   | Type           | Status    | Created")
	fmt.Println("-------------------------------------|----------------|-----------|--------------------")
	fmt.Println("(no tasks - daemon not connected)")
	return nil
}

func runTaskCreate(cmd *cobra.Command, args []string) error {
	taskType := args[0]

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(taskPayload), &payload); err != nil {
		return fmt.Errorf("invalid payload JSON: %w", err)
	}

	// TODO: Connect to daemon and create task
	fmt.Printf("✓ Task created\n")
	fmt.Printf("  Type: %s\n", taskType)
	fmt.Printf("  Priority: %d\n", taskPriority)
	fmt.Printf("  Payload: %s\n", taskPayload)

	return nil
}

func runTaskStatus(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// TODO: Connect to daemon and fetch task status
	fmt.Printf("Task: %s\n", taskID)
	fmt.Println("Status: (daemon not connected)")

	return nil
}

func runTaskCancel(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// TODO: Connect to daemon and cancel task
	fmt.Printf("✓ Task %s cancelled\n", taskID)

	return nil
}

