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
	rootCmd.AddCommand(logCmd)
	logCmd.AddCommand(logListCmd)
	logCmd.AddCommand(logSearchCmd)
	logCmd.AddCommand(logTailCmd)
	logCmd.AddCommand(logStatsCmd)
	logCmd.AddCommand(logParserCmd)
	logParserCmd.AddCommand(logParserListCmd)

	// Flags
	logListCmd.Flags().StringP("level", "l", "", "filter by level (trace, debug, info, warning, error, fatal)")
	logListCmd.Flags().StringP("service", "s", "", "filter by service name")
	logListCmd.Flags().StringP("source", "", "", "filter by source")
	logListCmd.Flags().StringP("trace-id", "t", "", "filter by trace ID")
	logListCmd.Flags().DurationP("since", "", time.Hour, "show logs since duration ago")
	logListCmd.Flags().IntP("limit", "n", 50, "limit number of results")

	logSearchCmd.Flags().DurationP("since", "", time.Hour, "search logs since duration ago")
	logSearchCmd.Flags().IntP("limit", "n", 50, "limit number of results")

	logTailCmd.Flags().StringP("level", "l", "", "filter by level")
	logTailCmd.Flags().StringP("service", "s", "", "filter by service name")

	logStatsCmd.Flags().DurationP("since", "", time.Hour, "stats for duration")
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "View and search logs",
	Long:  `View, search, and analyze aggregated logs.`,
}

var logListCmd = &cobra.Command{
	Use:   "list",
	Short: "List log entries",
	RunE:  runLogList,
}

var logSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search logs",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogSearch,
}

var logTailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Tail logs in real-time",
	RunE:  runLogTail,
}

var logStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show log statistics",
	RunE:  runLogStats,
}

var logParserCmd = &cobra.Command{
	Use:   "parser",
	Short: "Manage log parsers",
}

var logParserListCmd = &cobra.Command{
	Use:   "list",
	Short: "List log parsers",
	RunE:  runLogParserList,
}

func runLogList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	level, _ := cmd.Flags().GetString("level")
	service, _ := cmd.Flags().GetString("service")
	source, _ := cmd.Flags().GetString("source")
	traceID, _ := cmd.Flags().GetString("trace-id")
	since, _ := cmd.Flags().GetDuration("since")
	limit, _ := cmd.Flags().GetInt("limit")

	params := map[string]interface{}{
		"level":        level,
		"service_name": service,
		"source":       source,
		"trace_id":     traceID,
		"start_time":   time.Now().Add(-since).Format(time.RFC3339),
		"limit":        limit,
	}

	ctx := context.Background()
	resp, err := client.Call(ctx, "log.list", params)
	if err != nil {
		return fmt.Errorf("failed to list logs: %w", err)
	}

	logs, ok := resp["logs"].([]interface{})
	if !ok || len(logs) == 0 {
		fmt.Println("No logs found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIME\tLEVEL\tSERVICE\tMESSAGE")
	fmt.Fprintln(w, "----\t-----\t-------\t-------")

	for _, l := range logs {
		log := l.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			logFormatTime(getString(log, "timestamp")),
			getLevelIcon(getString(log, "level")),
			getString(log, "service_name"),
			truncateString(getString(log, "message"), 60),
		)
	}
	w.Flush()
	return nil
}

func runLogSearch(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	since, _ := cmd.Flags().GetDuration("since")
	limit, _ := cmd.Flags().GetInt("limit")

	params := map[string]interface{}{
		"query":      args[0],
		"start_time": time.Now().Add(-since).Format(time.RFC3339),
		"limit":      limit,
	}

	ctx := context.Background()
	resp, err := client.Call(ctx, "log.search", params)
	if err != nil {
		return fmt.Errorf("failed to search logs: %w", err)
	}

	logs, ok := resp["logs"].([]interface{})
	if !ok || len(logs) == 0 {
		fmt.Println("No logs found matching query.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIME\tLEVEL\tSERVICE\tMESSAGE")
	fmt.Fprintln(w, "----\t-----\t-------\t-------")

	for _, l := range logs {
		log := l.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			logFormatTime(getString(log, "timestamp")),
			getLevelIcon(getString(log, "level")),
			getString(log, "service_name"),
			truncateString(getString(log, "message"), 60),
		)
	}
	w.Flush()
	return nil
}

func runLogTail(cmd *cobra.Command, args []string) error {
	fmt.Println("Log tailing requires a running daemon with streaming support.")
	fmt.Println("Use 'forge log list --since 1m' to see recent logs.")
	return nil
}

func runLogStats(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	since, _ := cmd.Flags().GetDuration("since")
	params := map[string]interface{}{
		"start_time": time.Now().Add(-since).Format(time.RFC3339),
		"end_time":   time.Now().Format(time.RFC3339),
	}

	ctx := context.Background()
	resp, err := client.Call(ctx, "log.stats", params)
	if err != nil {
		return fmt.Errorf("failed to get log stats: %w", err)
	}

	fmt.Println("=== Log Statistics ===")
	fmt.Printf("Total Entries:  %v\n", resp["total_count"])
	fmt.Printf("Earliest:       %s\n", getString(resp, "earliest_timestamp"))
	fmt.Printf("Latest:         %s\n", getString(resp, "latest_timestamp"))

	if byLevel, ok := resp["by_level"].(map[string]interface{}); ok {
		fmt.Println("\nBy Level:")
		for level, count := range byLevel {
			fmt.Printf("  %s: %v\n", getLevelIcon(level), count)
		}
	}

	if byService, ok := resp["by_service"].(map[string]interface{}); ok {
		fmt.Println("\nBy Service:")
		for service, count := range byService {
			fmt.Printf("  %s: %v\n", service, count)
		}
	}

	return nil
}

func runLogParserList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "log.parser.list", nil)
	if err != nil {
		return fmt.Errorf("failed to list parsers: %w", err)
	}

	parsers, ok := resp["parsers"].([]interface{})
	if !ok || len(parsers) == 0 {
		fmt.Println("No log parsers configured.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tENABLED\tSOURCE FILTER")
	fmt.Fprintln(w, "--\t----\t----\t-------\t-------------")

	for _, p := range parsers {
		parser := p.(map[string]interface{})
		enabled := "✗"
		if e, ok := parser["enabled"].(bool); ok && e {
			enabled = "✓"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			traceTruncateID(getString(parser, "id")),
			getString(parser, "name"),
			getString(parser, "type"),
			enabled,
			getString(parser, "source_filter"),
		)
	}
	w.Flush()
	return nil
}

// Helper functions for log CLI
func logFormatTime(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("15:04:05.000")
}

func getLevelIcon(level string) string {
	switch level {
	case "trace":
		return "TRACE"
	case "debug":
		return "DEBUG"
	case "info":
		return "INFO"
	case "warning", "warn":
		return "WARN"
	case "error":
		return "ERROR"
	case "fatal":
		return "FATAL"
	default:
		return level
	}
}
