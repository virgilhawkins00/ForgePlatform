package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/forge-platform/forge/internal/adapters/tui"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:     "ui",
	Aliases: []string{"tui"},
	Short:   "Open the terminal user interface",
	Long: `Open the interactive terminal user interface (TUI).

The TUI provides:
  • Real-time metric dashboards
  • Log viewer with filtering
  • Task queue management
  • Plugin management
  • AI chat interface`,
	RunE: runUI,
}

var (
	uiTheme string
)

func init() {
	uiCmd.Flags().StringVar(&uiTheme, "theme", "dark", "UI theme (dark, light)")
}

func runUI(cmd *cobra.Command, args []string) error {
	// Create the TUI model
	model := tui.NewModel()

	// Create and run the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

