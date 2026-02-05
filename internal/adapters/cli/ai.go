package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Interact with the AI assistant",
	Long:  `Chat with the local LLM assistant and use AI-powered tools.`,
}

var aiChatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long:  `Start an interactive chat session with the AI assistant.`,
	RunE:  runAIChat,
}

var aiAskCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask a single question",
	Long:  `Ask a single question to the AI assistant.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAIAsk,
}

var aiModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models",
	Long:  `List all available LLM models.`,
	RunE:  runAIModels,
}

var aiAnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze system metrics with AI",
	Long:  `Use AI to analyze recent system metrics and logs.`,
	RunE:  runAIAnalyze,
}

var aiExplainCmd = &cobra.Command{
	Use:   "explain [metric]",
	Short: "Explain metric behavior or anomalies",
	Long:  `Use AI to explain the behavior of a specific metric or detected anomalies.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAIExplain,
}

var aiSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Get AI-powered optimization suggestions",
	Long:  `Use AI to suggest optimizations based on current system state.`,
	RunE:  runAISuggest,
}

var aiAutomateCmd = &cobra.Command{
	Use:   "automate [description]",
	Short: "Create automation rules from natural language",
	Long:  `Use AI to create automation rules from a natural language description.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAIAutomate,
}

var (
	aiModel       string
	aiTemperature float64
	aiTimeRange   string
	aiMetricName  string
	aiOutputJSON  bool
)

func init() {
	aiCmd.AddCommand(aiChatCmd)
	aiCmd.AddCommand(aiAskCmd)
	aiCmd.AddCommand(aiModelsCmd)
	aiCmd.AddCommand(aiAnalyzeCmd)
	aiCmd.AddCommand(aiExplainCmd)
	aiCmd.AddCommand(aiSuggestCmd)
	aiCmd.AddCommand(aiAutomateCmd)

	// Global AI flags
	aiCmd.PersistentFlags().StringVar(&aiModel, "model", "llama3.2", "LLM model to use")
	aiCmd.PersistentFlags().Float64Var(&aiTemperature, "temperature", 0.7, "Temperature for generation")
	aiCmd.PersistentFlags().BoolVar(&aiOutputJSON, "json", false, "Output results as JSON")

	// Analyze flags
	aiAnalyzeCmd.Flags().StringVar(&aiTimeRange, "range", "1h", "Time range to analyze")

	// Explain flags
	aiExplainCmd.Flags().StringVar(&aiMetricName, "metric", "", "Specific metric to explain")
	aiExplainCmd.Flags().StringVar(&aiTimeRange, "range", "1h", "Time range to analyze")

	// Suggest flags
	aiSuggestCmd.Flags().StringVar(&aiTimeRange, "range", "24h", "Time range to analyze")
}

func runAIChat(cmd *cobra.Command, args []string) error {
	fmt.Println("🤖 Forge AI Assistant")
	fmt.Printf("   Model: %s\n", aiModel)
	fmt.Println("   Type 'exit' or 'quit' to end the session")
	fmt.Println("   Type 'clear' to clear conversation history")
	fmt.Println()

	client, err := newDaemonClient()
	if err != nil {
		fmt.Println("⚠️  Daemon not connected. Running in offline mode.")
		fmt.Println("   Start the daemon with: forge start")
		fmt.Println()
	}

	reader := bufio.NewReader(os.Stdin)
	var history []string

	for {
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return nil
		case "clear":
			history = nil
			fmt.Println("Conversation cleared.")
			continue
		}

		history = append(history, input)

		// Try to get response from daemon
		if client != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			resp, err := client.Call(ctx, "ai.chat", map[string]interface{}{
				"message":     input,
				"model":       aiModel,
				"temperature": aiTemperature,
				"history":     history,
			})
			cancel()

			if err == nil && resp != nil {
				if content, ok := resp["content"].(string); ok {
					fmt.Println()
					fmt.Printf("Assistant: %s\n", content)
					fmt.Println()
					history = append(history, content)
					continue
				}
			}
		}

		fmt.Println()
		fmt.Println("Assistant: (AI provider not connected - run 'forge start' first)")
		fmt.Println()
	}
}

func runAIAsk(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	fmt.Printf("Question: %s\n\n", question)

	client, err := newDaemonClient()
	if err != nil {
		fmt.Println("Answer: (daemon not connected - run 'forge start' first)")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "ai.ask", map[string]interface{}{
		"question":    question,
		"model":       aiModel,
		"temperature": aiTemperature,
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}

	if content, ok := resp["content"].(string); ok {
		fmt.Printf("Answer: %s\n", content)
	} else {
		fmt.Println("Answer: (no response received)")
	}

	return nil
}

func runAIModels(cmd *cobra.Command, args []string) error {
	fmt.Println("Available Models:")
	fmt.Println()

	client, err := newDaemonClient()
	if err != nil {
		fmt.Println("  (Ollama not connected)")
		fmt.Println()
		fmt.Println("To install models, run:")
		fmt.Println("  ollama pull llama3.2")
		fmt.Println("  ollama pull mistral")
		fmt.Println("  ollama pull codellama")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "ai.models", nil)
	if err != nil {
		fmt.Println("  (failed to list models)")
		return nil
	}

	if models, ok := resp["models"].([]interface{}); ok {
		for _, m := range models {
			fmt.Printf("  • %v\n", m)
		}
	}

	return nil
}

func runAIAnalyze(cmd *cobra.Command, args []string) error {
	duration, err := time.ParseDuration(aiTimeRange)
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	fmt.Printf("🔍 Analyzing system metrics for the last %s...\n\n", aiTimeRange)

	client, err := newDaemonClient()
	if err != nil {
		fmt.Println("(daemon not connected - run 'forge start' first)")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "ai.analyze", map[string]interface{}{
		"time_range": duration.String(),
		"model":      aiModel,
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}

	if aiOutputJSON {
		output, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(output))
		return nil
	}

	// Display analysis results
	if issues, ok := resp["issues"].([]interface{}); ok && len(issues) > 0 {
		fmt.Println("⚠️  Issues Detected:")
		fmt.Println()
		for _, issue := range issues {
			if iss, ok := issue.(map[string]interface{}); ok {
				severity := iss["severity"]
				component := iss["component"]
				desc := iss["description"]
				suggestion := iss["suggestion"]

				icon := "ℹ️"
				switch severity {
				case "warning":
					icon = "⚠️"
				case "error":
					icon = "❌"
				case "critical":
					icon = "🔴"
				}

				fmt.Printf("%s [%s] %s\n", icon, component, desc)
				if suggestion != nil && suggestion != "" {
					fmt.Printf("   💡 %s\n", suggestion)
				}
				fmt.Println()
			}
		}
	} else {
		fmt.Println("✅ No issues detected. System looks healthy!")
	}

	if summary, ok := resp["summary"].(string); ok && summary != "" {
		fmt.Println("📊 Summary:")
		fmt.Println(summary)
	}

	return nil
}

func runAIExplain(cmd *cobra.Command, args []string) error {
	var metric string
	if len(args) > 0 {
		metric = args[0]
	} else if aiMetricName != "" {
		metric = aiMetricName
	}

	duration, err := time.ParseDuration(aiTimeRange)
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	if metric != "" {
		fmt.Printf("🔬 Explaining behavior of '%s' for the last %s...\n\n", metric, aiTimeRange)
	} else {
		fmt.Printf("🔬 Explaining system behavior for the last %s...\n\n", aiTimeRange)
	}

	client, err := newDaemonClient()
	if err != nil {
		fmt.Println("(daemon not connected - run 'forge start' first)")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "ai.explain", map[string]interface{}{
		"metric":     metric,
		"time_range": duration.String(),
		"model":      aiModel,
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}

	if aiOutputJSON {
		output, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(output))
		return nil
	}

	if explanation, ok := resp["explanation"].(string); ok {
		fmt.Println(explanation)
	}

	return nil
}

func runAISuggest(cmd *cobra.Command, args []string) error {
	duration, err := time.ParseDuration(aiTimeRange)
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	fmt.Printf("💡 Generating optimization suggestions based on last %s...\n\n", aiTimeRange)

	client, err := newDaemonClient()
	if err != nil {
		fmt.Println("(daemon not connected - run 'forge start' first)")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "ai.suggest", map[string]interface{}{
		"time_range": duration.String(),
		"model":      aiModel,
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}

	if aiOutputJSON {
		output, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(output))
		return nil
	}

	if suggestions, ok := resp["suggestions"].([]interface{}); ok {
		for i, s := range suggestions {
			if sug, ok := s.(map[string]interface{}); ok {
				title := sug["title"]
				desc := sug["description"]
				impact := sug["impact"]
				effort := sug["effort"]

				fmt.Printf("%d. %s\n", i+1, title)
				fmt.Printf("   %s\n", desc)
				if impact != nil {
					fmt.Printf("   Impact: %s | Effort: %s\n", impact, effort)
				}
				fmt.Println()
			}
		}
	}

	return nil
}

func runAIAutomate(cmd *cobra.Command, args []string) error {
	description := strings.Join(args, " ")

	fmt.Printf("🤖 Creating automation rule from: \"%s\"\n\n", description)

	client, err := newDaemonClient()
	if err != nil {
		fmt.Println("(daemon not connected - run 'forge start' first)")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "ai.automate", map[string]interface{}{
		"description": description,
		"model":       aiModel,
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}

	if aiOutputJSON {
		output, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(output))
		return nil
	}

	if rule, ok := resp["rule"].(map[string]interface{}); ok {
		fmt.Println("📋 Generated Automation Rule:")
		fmt.Println()

		if name, ok := rule["name"].(string); ok {
			fmt.Printf("Name: %s\n", name)
		}
		if trigger, ok := rule["trigger"].(string); ok {
			fmt.Printf("Trigger: %s\n", trigger)
		}
		if condition, ok := rule["condition"].(string); ok {
			fmt.Printf("Condition: %s\n", condition)
		}
		if action, ok := rule["action"].(string); ok {
			fmt.Printf("Action: %s\n", action)
		}

		fmt.Println()
		fmt.Println("To apply this rule, confirm with: forge workflow apply --from-ai")
	}

	return nil
}
