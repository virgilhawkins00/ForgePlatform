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
	rootCmd.AddCommand(traceCmd)
	traceCmd.AddCommand(traceListCmd)
	traceCmd.AddCommand(traceGetCmd)
	traceCmd.AddCommand(traceSpansCmd)
	traceCmd.AddCommand(traceServiceMapCmd)
	traceCmd.AddCommand(traceStatsCmd)

	// Flags
	traceListCmd.Flags().StringP("service", "s", "", "filter by service name")
	traceListCmd.Flags().StringP("status", "", "", "filter by status (ok, error)")
	traceListCmd.Flags().DurationP("since", "", 24*time.Hour, "show traces since duration ago")
	traceListCmd.Flags().IntP("limit", "n", 20, "limit number of results")

	traceServiceMapCmd.Flags().DurationP("since", "", 24*time.Hour, "time range for service map")
}

var traceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Manage distributed traces",
	Long:  `View and analyze distributed traces for debugging and performance analysis.`,
}

var traceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List traces",
	RunE:  runTraceList,
}

var traceGetCmd = &cobra.Command{
	Use:   "get <trace-id>",
	Short: "Get trace details",
	Args:  cobra.ExactArgs(1),
	RunE:  runTraceGet,
}

var traceSpansCmd = &cobra.Command{
	Use:   "spans <trace-id>",
	Short: "List spans in a trace",
	Args:  cobra.ExactArgs(1),
	RunE:  runTraceSpans,
}

var traceServiceMapCmd = &cobra.Command{
	Use:   "service-map",
	Short: "Show service dependency map",
	RunE:  runTraceServiceMap,
}

var traceStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show tracing statistics",
	RunE:  runTraceStats,
}

func runTraceList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	service, _ := cmd.Flags().GetString("service")
	status, _ := cmd.Flags().GetString("status")
	since, _ := cmd.Flags().GetDuration("since")
	limit, _ := cmd.Flags().GetInt("limit")

	params := map[string]interface{}{
		"service_name": service,
		"status":       status,
		"start_time":   time.Now().Add(-since).Format(time.RFC3339),
		"limit":        limit,
	}

	ctx := context.Background()
	resp, err := client.Call(ctx, "trace.list", params)
	if err != nil {
		return fmt.Errorf("failed to list traces: %w", err)
	}

	traces, ok := resp["traces"].([]interface{})
	if !ok || len(traces) == 0 {
		fmt.Println("No traces found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TRACE ID\tSERVICE\tNAME\tDURATION\tSPANS\tSTATUS\tSTARTED")
	fmt.Fprintln(w, "--------\t-------\t----\t--------\t-----\t------\t-------")

	for _, t := range traces {
		trace := t.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\t%s\t%s\n",
			traceTruncateID(getString(trace, "trace_id")),
			getString(trace, "service_name"),
			truncateString(getString(trace, "name"), 30),
			getString(trace, "duration"),
			trace["span_count"],
			getStatusIcon(getString(trace, "status")),
			traceFormatTime(getString(trace, "start_time")),
		)
	}
	w.Flush()
	return nil
}

func runTraceGet(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "trace.get", map[string]interface{}{"trace_id": args[0]})
	if err != nil {
		return fmt.Errorf("failed to get trace: %w", err)
	}

	trace, ok := resp["trace"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("trace not found")
	}

	fmt.Printf("Trace ID:     %s\n", getString(trace, "trace_id"))
	fmt.Printf("Service:      %s\n", getString(trace, "service_name"))
	fmt.Printf("Name:         %s\n", getString(trace, "name"))
	fmt.Printf("Status:       %s\n", getStatusIcon(getString(trace, "status")))
	fmt.Printf("Duration:     %s\n", getString(trace, "duration"))
	fmt.Printf("Span Count:   %v\n", trace["span_count"])
	fmt.Printf("Error Count:  %v\n", trace["error_count"])
	fmt.Printf("Started At:   %s\n", getString(trace, "start_time"))

	return nil
}

func runTraceSpans(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "trace.spans", map[string]interface{}{"trace_id": args[0]})
	if err != nil {
		return fmt.Errorf("failed to get spans: %w", err)
	}

	spans, ok := resp["spans"].([]interface{})
	if !ok || len(spans) == 0 {
		fmt.Println("No spans found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SPAN ID\tNAME\tKIND\tDURATION\tSTATUS\tSERVICE")
	fmt.Fprintln(w, "-------\t----\t----\t--------\t------\t-------")

	for _, s := range spans {
		span := s.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			traceTruncateID(getString(span, "span_id")),
			truncateString(getString(span, "name"), 30),
			getString(span, "kind"),
			getString(span, "duration"),
			getStatusIcon(getString(span, "status")),
			getString(span, "service_name"),
		)
	}
	w.Flush()
	return nil
}

func runTraceServiceMap(cmd *cobra.Command, args []string) error {
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
	resp, err := client.Call(ctx, "trace.service-map", params)
	if err != nil {
		return fmt.Errorf("failed to get service map: %w", err)
	}

	nodes, ok := resp["nodes"].([]interface{})
	if !ok || len(nodes) == 0 {
		fmt.Println("No services found in traces.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tSPAN COUNT\tERROR COUNT\tAVG DURATION\tDEPENDENCIES")
	fmt.Fprintln(w, "-------\t----------\t-----------\t------------\t------------")

	for _, n := range nodes {
		node := n.(map[string]interface{})
		deps := ""
		if d, ok := node["dependencies"].([]interface{}); ok {
			for i, dep := range d {
				if i > 0 {
					deps += ", "
				}
				deps += dep.(string)
			}
		}
		if deps == "" {
			deps = "-"
		}
		fmt.Fprintf(w, "%s\t%v\t%v\t%.2fms\t%s\n",
			getString(node, "service_name"),
			node["span_count"],
			node["error_count"],
			node["avg_duration_ms"],
			deps,
		)
	}
	w.Flush()
	return nil
}

func runTraceStats(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "trace.stats", nil)
	if err != nil {
		return fmt.Errorf("failed to get trace stats: %w", err)
	}

	fmt.Println("=== Trace Statistics ===")
	fmt.Printf("Active Traces: %v\n", resp["active_traces"])
	return nil
}

// Helper functions for trace CLI
func traceTruncateID(id string) string {
	if len(id) > 12 {
		return id[:12] + "..."
	}
	return id
}

func traceFormatTime(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("15:04:05")
}

func truncateString(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

func getStatusIcon(status string) string {
	switch status {
	case "ok":
		return "✓ ok"
	case "error":
		return "✗ error"
	default:
		return status
	}
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

