package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var alertCmd = &cobra.Command{
	Use:   "alert",
	Short: "Manage alerts and alert rules",
	Long:  `Create, list, and manage alert rules and view alert history.`,
}

var alertRuleCmd = &cobra.Command{
	Use:   "rule",
	Short: "Manage alert rules",
}

var alertRuleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all alert rules",
	RunE:  runAlertRuleList,
}

var alertRuleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new alert rule",
	RunE:  runAlertRuleCreate,
}

var alertRuleDeleteCmd = &cobra.Command{
	Use:   "delete <rule-id>",
	Short: "Delete an alert rule",
	Args:  cobra.ExactArgs(1),
	RunE:  runAlertRuleDelete,
}

var alertListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active alerts",
	RunE:  runAlertList,
}

var alertHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "View alert history",
	RunE:  runAlertHistory,
}

var alertAckCmd = &cobra.Command{
	Use:   "ack <alert-id>",
	Short: "Acknowledge an alert",
	Args:  cobra.ExactArgs(1),
	RunE:  runAlertAck,
}

var alertSilenceCmd = &cobra.Command{
	Use:   "silence",
	Short: "Manage silences",
}

var alertSilenceCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new silence",
	RunE:  runAlertSilenceCreate,
}

var alertSilenceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active silences",
	RunE:  runAlertSilenceList,
}

var alertChannelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage notification channels",
}

var alertChannelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List notification channels",
	RunE:  runAlertChannelList,
}

func init() {
	// Rule commands
	alertRuleCreateCmd.Flags().String("name", "", "Rule name (required)")
	alertRuleCreateCmd.Flags().String("metric", "", "Metric name to monitor (required)")
	alertRuleCreateCmd.Flags().String("condition", "threshold_above", "Condition type")
	alertRuleCreateCmd.Flags().Float64("threshold", 0, "Threshold value")
	alertRuleCreateCmd.Flags().String("severity", "warning", "Alert severity (info, warning, critical)")
	alertRuleCreateCmd.Flags().Duration("duration", time.Minute, "How long condition must be true")
	alertRuleCreateCmd.Flags().Duration("interval", time.Minute, "Evaluation interval")

	alertRuleCmd.AddCommand(alertRuleListCmd, alertRuleCreateCmd, alertRuleDeleteCmd)

	// Silence commands
	alertSilenceCreateCmd.Flags().StringToString("matchers", nil, "Label matchers (key=value)")
	alertSilenceCreateCmd.Flags().Duration("duration", time.Hour, "Silence duration")
	alertSilenceCreateCmd.Flags().String("comment", "", "Comment for the silence")

	alertSilenceCmd.AddCommand(alertSilenceCreateCmd, alertSilenceListCmd)

	// Channel commands
	alertChannelCmd.AddCommand(alertChannelListCmd)

	// Ack command
	alertAckCmd.Flags().String("comment", "", "Acknowledgement comment")

	// History command
	alertHistoryCmd.Flags().String("state", "", "Filter by state")
	alertHistoryCmd.Flags().String("severity", "", "Filter by severity")
	alertHistoryCmd.Flags().Int("limit", 50, "Maximum number of alerts to show")

	// Add all subcommands
	alertCmd.AddCommand(alertRuleCmd, alertListCmd, alertHistoryCmd, alertAckCmd, alertSilenceCmd, alertChannelCmd)
	rootCmd.AddCommand(alertCmd)
}

func runAlertRuleList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "alert.rule.list", nil)
	if err != nil {
		return fmt.Errorf("failed to list alert rules: %w", err)
	}

	rules, ok := resp["rules"].([]interface{})
	if !ok || len(rules) == 0 {
		fmt.Println("No alert rules found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tMETRIC\tCONDITION\tTHRESHOLD\tSEVERITY\tENABLED")
	fmt.Fprintln(w, "--\t----\t------\t---------\t---------\t--------\t-------")

	for _, r := range rules {
		rule := r.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.2f\t%s\t%v\n",
			alertTruncateID(rule["id"].(string)),
			rule["name"],
			rule["metric_name"],
			rule["condition"],
			rule["threshold"],
			rule["severity"],
			rule["enabled"],
		)
	}
	w.Flush()
	return nil
}

func runAlertRuleCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	metric, _ := cmd.Flags().GetString("metric")
	condition, _ := cmd.Flags().GetString("condition")
	threshold, _ := cmd.Flags().GetFloat64("threshold")
	severity, _ := cmd.Flags().GetString("severity")
	duration, _ := cmd.Flags().GetDuration("duration")
	interval, _ := cmd.Flags().GetDuration("interval")

	if name == "" || metric == "" {
		return fmt.Errorf("--name and --metric are required")
	}

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	params := map[string]interface{}{
		"name":        name,
		"metric_name": metric,
		"condition":   condition,
		"threshold":   threshold,
		"severity":    severity,
		"duration":    duration.String(),
		"interval":    interval.String(),
	}

	resp, err := client.Call(ctx, "alert.rule.create", params)
	if err != nil {
		return fmt.Errorf("failed to create alert rule: %w", err)
	}

	fmt.Printf("✅ Alert rule created: %s (ID: %s)\n", name, resp["id"])
	return nil
}

func runAlertRuleDelete(cmd *cobra.Command, args []string) error {
	ruleID := args[0]

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	_, err = client.Call(ctx, "alert.rule.delete", map[string]interface{}{"id": ruleID})
	if err != nil {
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	fmt.Printf("✅ Alert rule deleted: %s\n", ruleID)
	return nil
}

func runAlertList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "alert.list.active", nil)
	if err != nil {
		return fmt.Errorf("failed to list alerts: %w", err)
	}

	alerts, ok := resp["alerts"].([]interface{})
	if !ok || len(alerts) == 0 {
		fmt.Println("No active alerts.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tRULE\tSTATE\tSEVERITY\tVALUE\tSTARTED")
	fmt.Fprintln(w, "--\t----\t-----\t--------\t-----\t-------")

	for _, a := range alerts {
		alert := a.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.2f\t%s\n",
			alertTruncateID(alert["id"].(string)),
			alert["rule_name"],
			getStateIcon(alert["state"].(string)),
			alert["severity"],
			alert["value"],
			alertFormatTime(alert["starts_at"].(string)),
		)
	}
	w.Flush()
	return nil
}

func runAlertHistory(cmd *cobra.Command, args []string) error {
	state, _ := cmd.Flags().GetString("state")
	severity, _ := cmd.Flags().GetString("severity")
	limit, _ := cmd.Flags().GetInt("limit")

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	params := map[string]interface{}{
		"limit": limit,
	}
	if state != "" {
		params["state"] = state
	}
	if severity != "" {
		params["severity"] = severity
	}

	resp, err := client.Call(ctx, "alert.history", params)
	if err != nil {
		return fmt.Errorf("failed to get alert history: %w", err)
	}

	alerts, ok := resp["alerts"].([]interface{})
	if !ok || len(alerts) == 0 {
		fmt.Println("No alert history found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tRULE\tSTATE\tSEVERITY\tSTARTED\tENDED")
	fmt.Fprintln(w, "--\t----\t-----\t--------\t-------\t-----")

	for _, a := range alerts {
		alert := a.(map[string]interface{})
		endedAt := "-"
		if alert["ends_at"] != nil {
			endedAt = alertFormatTime(alert["ends_at"].(string))
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			alertTruncateID(alert["id"].(string)),
			alert["rule_name"],
			getStateIcon(alert["state"].(string)),
			alert["severity"],
			alertFormatTime(alert["starts_at"].(string)),
			endedAt,
		)
	}
	w.Flush()
	return nil
}

func runAlertAck(cmd *cobra.Command, args []string) error {
	alertID := args[0]
	comment, _ := cmd.Flags().GetString("comment")

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	params := map[string]interface{}{
		"id":      alertID,
		"comment": comment,
	}

	_, err = client.Call(ctx, "alert.ack", params)
	if err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	fmt.Printf("✅ Alert acknowledged: %s\n", alertID)
	return nil
}

func runAlertSilenceCreate(cmd *cobra.Command, args []string) error {
	matchers, _ := cmd.Flags().GetStringToString("matchers")
	duration, _ := cmd.Flags().GetDuration("duration")
	comment, _ := cmd.Flags().GetString("comment")

	if len(matchers) == 0 {
		return fmt.Errorf("--matchers is required")
	}

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	params := map[string]interface{}{
		"matchers": matchers,
		"duration": duration.String(),
		"comment":  comment,
	}

	resp, err := client.Call(ctx, "alert.silence.create", params)
	if err != nil {
		return fmt.Errorf("failed to create silence: %w", err)
	}

	fmt.Printf("✅ Silence created (ID: %s)\n", resp["id"])
	return nil
}

func runAlertSilenceList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "alert.silence.list", nil)
	if err != nil {
		return fmt.Errorf("failed to list silences: %w", err)
	}

	silences, ok := resp["silences"].([]interface{})
	if !ok || len(silences) == 0 {
		fmt.Println("No active silences.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tMATCHERS\tSTARTS\tENDS\tCOMMENT")
	fmt.Fprintln(w, "--\t--------\t------\t----\t-------")

	for _, s := range silences {
		silence := s.(map[string]interface{})
		matchersJSON, _ := json.Marshal(silence["matchers"])
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			alertTruncateID(silence["id"].(string)),
			string(matchersJSON),
			alertFormatTime(silence["starts_at"].(string)),
			alertFormatTime(silence["ends_at"].(string)),
			silence["comment"],
		)
	}
	w.Flush()
	return nil
}

func runAlertChannelList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Call(ctx, "alert.channel.list", nil)
	if err != nil {
		return fmt.Errorf("failed to list channels: %w", err)
	}

	channels, ok := resp["channels"].([]interface{})
	if !ok || len(channels) == 0 {
		fmt.Println("No notification channels configured.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tENABLED")
	fmt.Fprintln(w, "--\t----\t----\t-------")

	for _, c := range channels {
		channel := c.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\n",
			alertTruncateID(channel["id"].(string)),
			channel["name"],
			channel["type"],
			channel["enabled"],
		)
	}
	w.Flush()
	return nil
}

func getStateIcon(state string) string {
	switch state {
	case "firing":
		return "🔥 firing"
	case "pending":
		return "⏳ pending"
	case "resolved":
		return "✅ resolved"
	case "silenced":
		return "🔇 silenced"
	case "acknowledged":
		return "👁 acked"
	default:
		return state
	}
}

func alertFormatTime(timeStr string) string {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}
	return t.Format("2006-01-02 15:04")
}

func alertTruncateID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

