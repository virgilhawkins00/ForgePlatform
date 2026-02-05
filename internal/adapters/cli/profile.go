package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileStartCmd)
	profileStartCmd.AddCommand(profileStartCPUCmd)
	profileStartCmd.AddCommand(profileStartHeapCmd)
	profileStartCmd.AddCommand(profileStartGoroutineCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileGetCmd)
	profileCmd.AddCommand(profileStopCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	profileCmd.AddCommand(profileStatsCmd)
	profileCmd.AddCommand(profileMemoryCmd)

	// Flags
	profileStartCPUCmd.Flags().DurationP("duration", "d", 30*time.Second, "profile duration")
	profileStartCPUCmd.Flags().StringP("name", "n", "", "profile name")
	profileStartCPUCmd.Flags().StringP("service", "s", "forge", "service name")

	profileStartHeapCmd.Flags().StringP("name", "n", "", "profile name")
	profileStartHeapCmd.Flags().StringP("service", "s", "forge", "service name")

	profileStartGoroutineCmd.Flags().StringP("name", "n", "", "profile name")
	profileStartGoroutineCmd.Flags().StringP("service", "s", "forge", "service name")

	profileListCmd.Flags().StringP("type", "t", "", "filter by type (cpu, heap, goroutine)")
	profileListCmd.Flags().IntP("limit", "n", 20, "limit number of results")
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiling",
	Long:  `Capture and analyze CPU, memory, and goroutine profiles.`,
}

var profileStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a profile capture",
}

var profileStartCPUCmd = &cobra.Command{
	Use:   "cpu",
	Short: "Start CPU profiling",
	RunE:  runProfileStartCPU,
}

var profileStartHeapCmd = &cobra.Command{
	Use:   "heap",
	Short: "Capture heap profile",
	RunE:  runProfileStartHeap,
}

var profileStartGoroutineCmd = &cobra.Command{
	Use:   "goroutine",
	Short: "Capture goroutine profile",
	RunE:  runProfileStartGoroutine,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List profiles",
	RunE:  runProfileList,
}

var profileGetCmd = &cobra.Command{
	Use:   "get <profile-id>",
	Short: "Get profile details",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileGet,
}

var profileStopCmd = &cobra.Command{
	Use:   "stop <profile-id>",
	Short: "Stop an active profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileStop,
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <profile-id>",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileDelete,
}

var profileStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show profiling statistics",
	RunE:  runProfileStats,
}

var profileMemoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Show current memory statistics",
	RunE:  runProfileMemory,
}

func runProfileStartCPU(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	duration, _ := cmd.Flags().GetDuration("duration")
	name, _ := cmd.Flags().GetString("name")
	service, _ := cmd.Flags().GetString("service")

	if name == "" {
		name = fmt.Sprintf("cpu-%s", time.Now().Format("20060102-150405"))
	}

	params := map[string]interface{}{
		"name":         name,
		"service_name": service,
		"duration":     duration.String(),
	}

	ctx := context.Background()
	resp, err := client.Call(ctx, "profile.start.cpu", params)
	if err != nil {
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}

	fmt.Printf("✓ Started CPU profile: %s\n", getString(resp, "id"))
	fmt.Printf("  Duration: %s\n", duration)
	fmt.Printf("  Will auto-stop after duration completes.\n")
	return nil
}

func runProfileStartHeap(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	name, _ := cmd.Flags().GetString("name")
	service, _ := cmd.Flags().GetString("service")

	if name == "" {
		name = fmt.Sprintf("heap-%s", time.Now().Format("20060102-150405"))
	}

	params := map[string]interface{}{
		"name":         name,
		"service_name": service,
	}

	ctx := context.Background()
	resp, err := client.Call(ctx, "profile.start.heap", params)
	if err != nil {
		return fmt.Errorf("failed to capture heap profile: %w", err)
	}

	fmt.Printf("✓ Captured heap profile: %s\n", getString(resp, "id"))
	fmt.Printf("  Size: %v bytes\n", resp["data_size"])
	return nil
}

func runProfileStartGoroutine(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	name, _ := cmd.Flags().GetString("name")
	service, _ := cmd.Flags().GetString("service")

	if name == "" {
		name = fmt.Sprintf("goroutine-%s", time.Now().Format("20060102-150405"))
	}

	params := map[string]interface{}{
		"name":         name,
		"service_name": service,
	}

	ctx := context.Background()
	resp, err := client.Call(ctx, "profile.start.goroutine", params)
	if err != nil {
		return fmt.Errorf("failed to capture goroutine profile: %w", err)
	}

	fmt.Printf("✓ Captured goroutine profile: %s\n", getString(resp, "id"))
	fmt.Printf("  Size: %v bytes\n", resp["data_size"])
	return nil
}

func runProfileList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	profileType, _ := cmd.Flags().GetString("type")
	limit, _ := cmd.Flags().GetInt("limit")

	params := map[string]interface{}{
		"type":  profileType,
		"limit": limit,
	}

	ctx := context.Background()
	resp, err := client.Call(ctx, "profile.list", params)
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	profiles, ok := resp["profiles"].([]interface{})
	if !ok || len(profiles) == 0 {
		fmt.Println("No profiles found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tSTATUS\tSIZE\tCREATED")
	fmt.Fprintln(w, "--\t----\t----\t------\t----\t-------")

	for _, p := range profiles {
		profile := p.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\t%s\n",
			traceTruncateID(getString(profile, "id")),
			truncateString(getString(profile, "name"), 25),
			getString(profile, "type"),
			getProfileStatusIcon(getString(profile, "status")),
			formatBytes(profile["data_size"]),
			profileFormatTime(getString(profile, "created_at")),
		)
	}
	w.Flush()
	return nil
}

func runProfileGet(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "profile.get", map[string]interface{}{"id": args[0]})
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	profile, ok := resp["profile"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("profile not found")
	}

	fmt.Printf("ID:           %s\n", getString(profile, "id"))
	fmt.Printf("Name:         %s\n", getString(profile, "name"))
	fmt.Printf("Type:         %s\n", getString(profile, "type"))
	fmt.Printf("Status:       %s\n", getProfileStatusIcon(getString(profile, "status")))
	fmt.Printf("Service:      %s\n", getString(profile, "service_name"))
	fmt.Printf("Duration:     %s\n", getString(profile, "duration"))
	fmt.Printf("Data Size:    %v\n", formatBytes(profile["data_size"]))
	fmt.Printf("File Path:    %s\n", getString(profile, "file_path"))
	fmt.Printf("Created At:   %s\n", getString(profile, "created_at"))

	return nil
}

func runProfileStop(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "profile.stop", map[string]interface{}{"id": args[0]})
	if err != nil {
		return fmt.Errorf("failed to stop profile: %w", err)
	}

	fmt.Printf("✓ Stopped profile: %s\n", getString(resp, "id"))
	fmt.Printf("  Size: %v\n", formatBytes(resp["data_size"]))
	return nil
}

func runProfileDelete(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	_, err = client.Call(ctx, "profile.delete", map[string]interface{}{"id": args[0]})
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	fmt.Printf("✓ Deleted profile: %s\n", args[0])
	return nil
}

func runProfileStats(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "profile.stats", nil)
	if err != nil {
		return fmt.Errorf("failed to get profile stats: %w", err)
	}

	fmt.Println("=== Profile Statistics ===")
	fmt.Printf("Active Profiles:  %v\n", resp["active_profiles"])
	fmt.Printf("Goroutines:       %v\n", resp["num_goroutine"])
	fmt.Printf("Heap Alloc:       %.2f MB\n", resp["heap_alloc_mb"])
	fmt.Printf("Sys Memory:       %.2f MB\n", resp["sys_mb"])
	fmt.Printf("GC Cycles:        %v\n", resp["num_gc"])

	return nil
}

func runProfileMemory(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "profile.memory", nil)
	if err != nil {
		return fmt.Errorf("failed to get memory stats: %w", err)
	}

	fmt.Println("=== Memory Statistics ===")
	fmt.Printf("Alloc:          %s\n", formatBytesUint(resp["alloc"]))
	fmt.Printf("Total Alloc:    %s\n", formatBytesUint(resp["total_alloc"]))
	fmt.Printf("Sys:            %s\n", formatBytesUint(resp["sys"]))
	fmt.Printf("Heap Alloc:     %s\n", formatBytesUint(resp["heap_alloc"]))
	fmt.Printf("Heap Sys:       %s\n", formatBytesUint(resp["heap_sys"]))
	fmt.Printf("Heap Idle:      %s\n", formatBytesUint(resp["heap_idle"]))
	fmt.Printf("Heap Inuse:     %s\n", formatBytesUint(resp["heap_inuse"]))
	fmt.Printf("Heap Released:  %s\n", formatBytesUint(resp["heap_released"]))
	fmt.Printf("Heap Objects:   %v\n", resp["heap_objects"])
	fmt.Printf("Stack Inuse:    %s\n", formatBytesUint(resp["stack_inuse"]))
	fmt.Printf("Stack Sys:      %s\n", formatBytesUint(resp["stack_sys"]))
	fmt.Printf("Num GC:         %v\n", resp["num_gc"])
	fmt.Printf("Num Goroutine:  %v\n", resp["num_goroutine"])

	return nil
}

// Helper functions for profile CLI
func profileFormatTime(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("01/02 15:04")
}

func getProfileStatusIcon(status string) string {
	switch status {
	case "pending":
		return "⏳ pending"
	case "capturing":
		return "🔴 capturing"
	case "completed":
		return "✓ completed"
	case "failed":
		return "✗ failed"
	default:
		return status
	}
}

func formatBytes(v interface{}) string {
	var bytes float64
	switch t := v.(type) {
	case float64:
		bytes = t
	case int:
		bytes = float64(t)
	case int64:
		bytes = float64(t)
	default:
		return fmt.Sprintf("%v", v)
	}

	if bytes < 1024 {
		return fmt.Sprintf("%.0f B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", bytes/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", bytes/1024/1024)
	}
	return fmt.Sprintf("%.2f GB", bytes/1024/1024/1024)
}

func formatBytesUint(v interface{}) string {
	var bytes uint64
	switch t := v.(type) {
	case float64:
		bytes = uint64(t)
	case int:
		bytes = uint64(t)
	case int64:
		bytes = uint64(t)
	case uint64:
		bytes = t
	default:
		return fmt.Sprintf("%v", v)
	}

	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/1024/1024)
	}
	return fmt.Sprintf("%.2f GB", float64(bytes)/1024/1024/1024)
}
