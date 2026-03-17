package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var metricCmd = &cobra.Command{
	Use:   "metric",
	Short: "Manage metrics in the TSDB",
	Long:  `Record, query, and manage time-series metrics.`,
}

var metricRecordCmd = &cobra.Command{
	Use:   "record [name] [value]",
	Short: "Record a metric",
	Long:  `Record a new metric value to the time-series database.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runMetricRecord,
}

var metricQueryCmd = &cobra.Command{
	Use:   "query [name]",
	Short: "Query metrics",
	Long:  `Query metrics from the time-series database.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runMetricQuery,
}

var metricListCmd = &cobra.Command{
	Use:   "list",
	Short: "List metric series",
	Long:  `List all unique metric series in the database.`,
	RunE:  runMetricList,
}

var metricStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show TSDB statistics",
	Long:  `Display statistics about the time-series database.`,
	RunE:  runMetricStats,
}

var metricDownsampleCmd = &cobra.Command{
	Use:   "downsample",
	Short: "Trigger downsampling of old metrics",
	Long: `Aggregate old raw metrics into lower resolution buckets.

Resolutions:
  - 1m: 1-minute buckets (retained for 30 days)
  - 1h: 1-hour buckets (retained for 1 year)
  - 1d: 1-day buckets (retained forever)

Example:
  forge metric downsample --older-than 7d --resolution 1m`,
	RunE: runMetricDownsample,
}

var metricAggregateCmd = &cobra.Command{
	Use:   "aggregate [name]",
	Short: "Query metrics with aggregation",
	Long: `Query metrics with time-bucket aggregation.

Aggregation types: avg, sum, min, max, count, first, last

Example:
  forge metric aggregate cpu.usage --agg avg --step 5m --start -1h`,
	Args: cobra.ExactArgs(1),
	RunE: runMetricAggregate,
}

var (
	metricTags       string
	metricType       string
	metricStart      string
	metricEnd        string
	metricInterval   string
	metricOlderThan  string
	metricResolution string
	metricAggType    string
	metricStep       string
)

func init() {
	metricCmd.AddCommand(metricRecordCmd)
	metricCmd.AddCommand(metricQueryCmd)
	metricCmd.AddCommand(metricListCmd)
	metricCmd.AddCommand(metricStatsCmd)
	metricCmd.AddCommand(metricDownsampleCmd)
	metricCmd.AddCommand(metricAggregateCmd)

	// Record flags
	metricRecordCmd.Flags().StringVar(&metricTags, "tags", "", "Metric tags (key=value,key2=value2)")
	metricRecordCmd.Flags().StringVar(&metricType, "type", "gauge", "Metric type (gauge, counter, histogram)")

	// Query flags
	metricQueryCmd.Flags().StringVar(&metricTags, "tags", "", "Filter by tags")
	metricQueryCmd.Flags().StringVar(&metricStart, "start", "-1h", "Start time (e.g., -1h, -24h, 2024-01-01)")
	metricQueryCmd.Flags().StringVar(&metricEnd, "end", "now", "End time")
	metricQueryCmd.Flags().StringVar(&metricInterval, "interval", "", "Aggregation interval (1m, 5m, 1h)")

	// Downsample flags
	metricDownsampleCmd.Flags().StringVar(&metricOlderThan, "older-than", "7d", "Age threshold for downsampling (e.g., 7d, 24h)")
	metricDownsampleCmd.Flags().StringVar(&metricResolution, "resolution", "1m", "Target resolution (1m, 1h, 1d)")

	// Aggregate flags
	metricAggregateCmd.Flags().StringVar(&metricAggType, "agg", "avg", "Aggregation type (avg, sum, min, max, count, first, last)")
	metricAggregateCmd.Flags().StringVar(&metricStep, "step", "1m", "Time bucket size (1m, 5m, 1h, 1d)")
	metricAggregateCmd.Flags().StringVar(&metricStart, "start", "-1h", "Start time")
	metricAggregateCmd.Flags().StringVar(&metricEnd, "end", "now", "End time")
	metricAggregateCmd.Flags().StringVar(&metricTags, "tags", "", "Filter by tags")
}

func runMetricRecord(cmd *cobra.Command, args []string) error {
	name := args[0]
	valueStr := args[1]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}

	tags := parseTags(metricTags)

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	params := map[string]interface{}{
		"name":  name,
		"value": value,
		"type":  metricType,
		"tags":  tags,
	}

	_, err = client.Call(cmd.Context(), "metric.record", params)
	if err != nil {
		return fmt.Errorf("failed to record metric: %w", err)
	}

	fmt.Printf("✓ Metric recorded\n")
	fmt.Printf("  Name: %s\n", name)
	fmt.Printf("  Value: %.2f\n", value)
	fmt.Printf("  Type: %s\n", metricType)
	if len(tags) > 0 {
		fmt.Printf("  Tags: %v\n", tags)
	}
	fmt.Printf("  Timestamp: %s\n", time.Now().Format(time.RFC3339))

	return nil
}

func runMetricQuery(cmd *cobra.Command, args []string) error {
	name := args[0]

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	start, err := parseTimeSpec(metricStart)
	if err != nil {
		return err
	}
	end, err := parseTimeSpec(metricEnd)
	if err != nil {
		return err
	}

	params := map[string]interface{}{
		"name":  name,
		"start": start.Format(time.RFC3339),
		"end":   end.Format(time.RFC3339),
		"tags":  parseTags(metricTags),
		"limit": 100, // default limit
	}

	resp, err := client.Call(cmd.Context(), "metric.query", params)
	if err != nil {
		return fmt.Errorf("failed to query metrics: %w", err)
	}

	fmt.Printf("Querying metric: %s\n", name)
	fmt.Printf("  Time range: %s to %s\n", metricStart, metricEnd)
	
	resMap, ok := resp.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response type")
	}

	if points, ok := resMap["points"].([]interface{}); ok {
		fmt.Printf("\nFound %d points:\n", len(points))
		for _, p := range points {
			pt := p.(map[string]interface{})
			fmt.Printf("  %s: %v\n", pt["timestamp"], pt["value"])
		}
	} else {
		fmt.Println("\nNo points found.")
	}

	return nil
}

func runMetricList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	resp, err := client.Call(cmd.Context(), "metric.list", nil)
	if err != nil {
		return fmt.Errorf("failed to list series: %w", err)
	}

	resMap, ok := resp.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response type")
	}

	fmt.Println("Metric Series:")
	if seriesList, ok := resMap["series"].([]interface{}); ok {
		for _, s := range seriesList {
			sv := s.(map[string]interface{})
			fmt.Printf("  %s\n", sv["name"])
			if tags, ok := sv["tags"].(map[string]interface{}); ok && len(tags) > 0 {
				fmt.Printf("    Tags: %v\n", tags)
			}
			fmt.Printf("    Range: %s - %s\n", sv["first_time"], sv["last_time"])
		}
	}

	return nil
}

func runMetricStats(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	resp, err := client.Call(cmd.Context(), "metric.stats", nil)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	resMap, ok := resp.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response type")
	}

	fmt.Println("TSDB Statistics:")
	fmt.Printf("  Total points: %v\n", resMap["TotalPoints"])
	fmt.Printf("  Total series: %v\n", resMap["TotalSeries"])
	fmt.Printf("  Storage space: %v bytes\n", resMap["StorageBytes"])
	fmt.Printf("  Time range: %v to %v\n", resMap["OldestPoint"], resMap["NewestPoint"])
	
	if agg, ok := resMap["AggregatedPoints"].(map[string]interface{}); ok {
		fmt.Println("  Aggregated points:")
		for res, count := range agg {
			fmt.Printf("    %s: %v\n", res, count)
		}
	}

	return nil
}

func runMetricDownsample(cmd *cobra.Command, args []string) error {
	olderThan, err := parseDuration(metricOlderThan)
	if err != nil {
		return fmt.Errorf("invalid --older-than value: %w", err)
	}

	// Validate resolution
	validResolutions := map[string]bool{"1m": true, "5m": true, "1h": true, "1d": true}
	if !validResolutions[metricResolution] {
		return fmt.Errorf("invalid resolution: %s (use 1m, 5m, 1h, or 1d)", metricResolution)
	}

	fmt.Printf("🔄 Triggering downsampling...\n")
	fmt.Printf("  Metrics older than: %s\n", metricOlderThan)
	fmt.Printf("  Target resolution: %s\n", metricResolution)
	fmt.Println("\n(daemon not connected - would aggregate old metrics)")

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	params := map[string]interface{}{
		"older_than": olderThan.String(),
		"resolution": metricResolution,
	}

	_, err = client.Call(cmd.Context(), "metric.downsample", params)
	if err != nil {
		return fmt.Errorf("failed to downsample metrics: %w", err)
	}

	fmt.Printf("✓ Downsampling started/completed for metrics older than %s to %s resolution.\n", metricOlderThan, metricResolution)
	return nil
}

func runMetricAggregate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Parse time range
	start, err := parseTimeSpec(metricStart)
	if err != nil {
		return fmt.Errorf("invalid --start value: %w", err)
	}

	end, err := parseTimeSpec(metricEnd)
	if err != nil {
		return fmt.Errorf("invalid --end value: %w", err)
	}

	// Validate aggregation type
	validAggs := map[string]bool{
		"avg": true, "sum": true, "min": true, "max": true,
		"count": true, "first": true, "last": true,
	}
	if !validAggs[metricAggType] {
		return fmt.Errorf("invalid aggregation type: %s", metricAggType)
	}

	fmt.Printf("📊 Aggregated Query: %s\n", name)
	fmt.Printf("  Aggregation: %s\n", metricAggType)
	fmt.Printf("  Step: %s\n", metricStep)
	fmt.Printf("  Time range: %s to %s\n", start.Format(time.RFC3339), end.Format(time.RFC3339))
	if metricTags != "" {
		fmt.Printf("  Tags filter: %s\n", metricTags)
	}
	fmt.Println("\n(daemon not connected)")

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	params := map[string]interface{}{
		"name":  name,
		"agg":   metricAggType,
		"step":  metricStep,
		"tags":  parseTags(metricTags),
		"start": start.Format(time.RFC3339),
		"end":   end.Format(time.RFC3339),
	}

	resp, err := client.Call(cmd.Context(), "metric.aggregate", params)
	if err != nil {
		return fmt.Errorf("failed to execute aggregate query: %w", err)
	}

	resMap, ok := resp.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response type")
	}

	if points, ok := resMap["points"].([]interface{}); ok {
		fmt.Printf("\nFound %d aggregated points:\n", len(points))
		for _, p := range points {
			pt := p.(map[string]interface{})
			fmt.Printf("  %s: %v\n", pt["timestamp"], pt[metricAggType])
		}
	} else {
		fmt.Println("\nNo points found.")
	}

	return nil
}

func parseTags(tagStr string) map[string]string {
	tags := make(map[string]string)
	if tagStr == "" {
		return tags
	}
	pairs := strings.Split(tagStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			tags[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return tags
}

// parseDuration parses duration strings like "7d", "24h", "1h30m"
func parseDuration(s string) (time.Duration, error) {
	// Handle day suffix (not supported by time.ParseDuration)
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

// parseTimeSpec parses time specifications like "-1h", "-24h", "now", or RFC3339
func parseTimeSpec(s string) (time.Time, error) {
	now := time.Now()

	if s == "now" {
		return now, nil
	}

	// Relative time (e.g., -1h, -24h, -7d)
	if strings.HasPrefix(s, "-") {
		dur, err := parseDuration(s[1:])
		if err != nil {
			return time.Time{}, err
		}
		return now.Add(-dur), nil
	}

	// Try RFC3339 format
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	// Try date only format
	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unrecognized time format: %s", s)
}

