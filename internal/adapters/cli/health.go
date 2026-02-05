// Package cli implements the command-line interface for Forge.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/forge-platform/forge/internal/adapters/daemon"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check system health",
	Long:  "Check the health status of the Forge platform and its components.",
	RunE:  runHealth,
}

var livenessCmd = &cobra.Command{
	Use:   "liveness",
	Short: "Check if the daemon is alive",
	RunE:  runLiveness,
}

var readinessCmd = &cobra.Command{
	Use:   "readiness",
	Short: "Check if the daemon is ready to accept requests",
	RunE:  runReadiness,
}

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Display system metrics",
	Long:  "Display system metrics including memory, goroutines, and GC statistics.",
	RunE:  runMetrics,
}

var (
	healthOutputJSON bool
)

func init() {
	healthCmd.AddCommand(livenessCmd)
	healthCmd.AddCommand(readinessCmd)
	healthCmd.AddCommand(metricsCmd)
	healthCmd.Flags().BoolVar(&healthOutputJSON, "json", false, "Output in JSON format")
	metricsCmd.Flags().BoolVar(&healthOutputJSON, "json", false, "Output in JSON format")
}

func runHealth(cmd *cobra.Command, args []string) error {
	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Call(context.Background(), "health", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if healthOutputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	// Display formatted output
	status, _ := resp["status"].(string)
	version, _ := resp["version"].(string)
	uptime, _ := resp["uptime"].(string)

	fmt.Printf("Status:  %s\n", colorStatus(status))
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Uptime:  %s\n", uptime)

	// Display components
	if components, ok := resp["components"].([]interface{}); ok && len(components) > 0 {
		fmt.Println("\nComponents:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tMESSAGE\tLATENCY")
		for _, c := range components {
			comp := c.(map[string]interface{})
			name, _ := comp["name"].(string)
			cstatus, _ := comp["status"].(string)
			message, _ := comp["message"].(string)
			latency, _ := comp["latency_ms"].(float64)
			fmt.Fprintf(w, "%s\t%s\t%s\t%.0fms\n", name, colorStatus(cstatus), message, latency)
		}
		w.Flush()
	}

	// Display system metrics
	if sys, ok := resp["system"].(map[string]interface{}); ok {
		fmt.Println("\nSystem:")
		goVersion, _ := sys["go_version"].(string)
		goroutines, _ := sys["goroutines"].(float64)
		cpus, _ := sys["cpus"].(float64)
		heapAlloc, _ := sys["heap_alloc"].(float64)
		fmt.Printf("  Go Version:  %s\n", goVersion)
		fmt.Printf("  Goroutines:  %.0f\n", goroutines)
		fmt.Printf("  CPUs:        %.0f\n", cpus)
		fmt.Printf("  Heap Alloc:  %.2f MB\n", heapAlloc/1024/1024)
	}

	return nil
}

func runLiveness(cmd *cobra.Command, args []string) error {
	client, err := daemon.NewClient("")
	if err != nil {
		fmt.Println("NOT ALIVE: Failed to connect to daemon")
		os.Exit(1)
	}
	defer client.Close()

	resp, err := client.Call(context.Background(), "health.liveness", nil)
	if err != nil {
		fmt.Println("NOT ALIVE:", err)
		os.Exit(1)
	}

	if alive, ok := resp["alive"].(bool); ok && alive {
		fmt.Println("ALIVE")
		return nil
	}
	fmt.Println("NOT ALIVE")
	os.Exit(1)
	return nil
}

func runReadiness(cmd *cobra.Command, args []string) error {
	client, err := daemon.NewClient("")
	if err != nil {
		fmt.Println("NOT READY: Failed to connect to daemon")
		os.Exit(1)
	}
	defer client.Close()

	resp, err := client.Call(context.Background(), "health.readiness", nil)
	if err != nil {
		fmt.Println("NOT READY:", err)
		os.Exit(1)
	}

	if ready, ok := resp["ready"].(bool); ok && ready {
		fmt.Println("READY")
		return nil
	}
	fmt.Println("NOT READY")
	os.Exit(1)
	return nil
}

func runMetrics(cmd *cobra.Command, args []string) error {
	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Call(context.Background(), "health.metrics", nil)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	if healthOutputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	// Display metrics in Prometheus-like format
	fmt.Println("# HELP forge_uptime_seconds Forge daemon uptime in seconds")
	fmt.Println("# TYPE forge_uptime_seconds gauge")
	if v, ok := resp["forge_uptime_seconds"].(float64); ok {
		fmt.Printf("forge_uptime_seconds %.2f\n\n", v)
	}

	fmt.Println("# HELP go_goroutines Number of goroutines")
	fmt.Println("# TYPE go_goroutines gauge")
	if v, ok := resp["go_goroutines"].(float64); ok {
		fmt.Printf("go_goroutines %.0f\n\n", v)
	}

	fmt.Println("# HELP go_memstats_alloc_bytes Bytes of allocated heap objects")
	fmt.Println("# TYPE go_memstats_alloc_bytes gauge")
	if v, ok := resp["go_memstats_alloc"].(float64); ok {
		fmt.Printf("go_memstats_alloc_bytes %.0f\n\n", v)
	}

	fmt.Println("# HELP go_memstats_heap_bytes Bytes of heap memory obtained from the OS")
	fmt.Println("# TYPE go_memstats_heap_bytes gauge")
	if v, ok := resp["go_memstats_heap"].(float64); ok {
		fmt.Printf("go_memstats_heap_bytes %.0f\n\n", v)
	}

	fmt.Println("# HELP go_gc_duration_ns Latest GC pause duration in nanoseconds")
	fmt.Println("# TYPE go_gc_duration_ns gauge")
	if v, ok := resp["go_gc_duration_ns"].(float64); ok {
		fmt.Printf("go_gc_duration_ns %.0f\n\n", v)
	}

	fmt.Println("# HELP go_gc_count Total number of GC cycles")
	fmt.Println("# TYPE go_gc_count counter")
	if v, ok := resp["go_gc_count"].(float64); ok {
		fmt.Printf("go_gc_count %.0f\n", v)
	}

	return nil
}

// colorStatus returns status with ANSI colors.
func colorStatus(status string) string {
	switch status {
	case "healthy":
		return "\033[32m" + status + "\033[0m" // Green
	case "degraded":
		return "\033[33m" + status + "\033[0m" // Yellow
	case "unhealthy":
		return "\033[31m" + status + "\033[0m" // Red
	default:
		return status
	}
}

