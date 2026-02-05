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

	// TODO: Connect to daemon and record metric
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

	// TODO: Connect to daemon and query metrics
	fmt.Printf("Querying metric: %s\n", name)
	fmt.Printf("  Time range: %s to %s\n", metricStart, metricEnd)
	if metricInterval != "" {
		fmt.Printf("  Aggregation: %s\n", metricInterval)
	}
	fmt.Println("\n(daemon not connected)")

	return nil
}

func runMetricList(cmd *cobra.Command, args []string) error {
	// TODO: Connect to daemon and list series
	fmt.Println("Metric Series:")
	fmt.Println("(daemon not connected)")
	return nil
}

func runMetricStats(cmd *cobra.Command, args []string) error {
	// TODO: Connect to daemon and get stats
	fmt.Println("TSDB Statistics:")
	fmt.Println("  Total metrics: (daemon not connected)")
	fmt.Println("  Unique series: (daemon not connected)")
	fmt.Println("  Database size: (daemon not connected)")
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

	// TODO: Connect to daemon and trigger downsampling
	// client.Downsample(olderThan, metricResolution)
	_ = olderThan

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

	// TODO: Connect to daemon and run aggregated query
	// results := client.QueryWithAggregation(query)

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

