package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/forge-platform/forge/internal/adapters/daemon"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/spf13/cobra"
)

// newDaemonClient creates a new daemon client connected to the default socket.
func newDaemonClient() (*daemon.Client, error) {
	forgeDir, err := getForgeDir()
	if err != nil {
		return nil, err
	}

	client, err := daemon.NewClient(forgeDir)
	if err != nil {
		return nil, err
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}

	return client, nil
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Forge daemon",
	Long: `Start the Forge background daemon process.

The daemon provides:
  • Task queue processing
  • Metric collection and storage
  • Plugin runtime management
  • AI agent orchestration
  • gRPC API over Unix socket`,
	RunE: runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Forge daemon",
	Long:  `Gracefully stop the running Forge daemon.`,
	RunE:  runStop,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long:  `Display the current status of the Forge daemon.`,
	RunE:  runStatus,
}

func runStart(cmd *cobra.Command, args []string) error {
	forgeDir, err := ensureForgeDir()
	if err != nil {
		return err
	}

	// Use default configuration from daemon package
	config := daemon.DefaultConfig(forgeDir)

	// Check if already running
	if _, err := os.Stat(config.SocketPath); err == nil {
		return fmt.Errorf("daemon already running (socket exists: %s)", config.SocketPath)
	}

	// Create a simple logger that prints to stdout
	logger := &simpleLogger{}

	server, err := daemon.NewServer(config, logger)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	fmt.Printf("🚀 Forge daemon started\n")
	fmt.Printf("   Socket: %s\n", config.SocketPath)
	fmt.Printf("   PID: %d\n", os.Getpid())
	fmt.Println("   Press Ctrl+C to stop")

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := server.Start(ctx); err != nil {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		fmt.Println("\n⏳ Shutting down gracefully...")
	case err := <-errCh:
		fmt.Printf("\n❌ Server error: %v\n", err)
		return err
	}

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = server.Stop(shutdownCtx)

	fmt.Println("✓ Daemon stopped")
	return nil
}

// simpleLogger is a basic logger implementation that prints to stdout.
type simpleLogger struct {
	prefix string
}

func (l *simpleLogger) Debug(msg string, args ...interface{}) {
	fmt.Printf("[DEBUG]%s %s %v\n", l.prefix, msg, args)
}

func (l *simpleLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("[INFO]%s %s %v\n", l.prefix, msg, args)
}

func (l *simpleLogger) Warn(msg string, args ...interface{}) {
	fmt.Printf("[WARN]%s %s %v\n", l.prefix, msg, args)
}

func (l *simpleLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("[ERROR]%s %s %v\n", l.prefix, msg, args)
}

func (l *simpleLogger) With(args ...interface{}) ports.Logger {
	return &simpleLogger{prefix: l.prefix + fmt.Sprintf(" %v", args)}
}

func runStop(cmd *cobra.Command, args []string) error {
	forgeDir, err := getForgeDir()
	if err != nil {
		return err
	}

	pidFile := filepath.Join(forgeDir, "forge.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("daemon not running (no PID file)")
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return fmt.Errorf("invalid PID file")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	fmt.Printf("✓ Sent stop signal to daemon (PID: %d)\n", pid)
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	forgeDir, err := getForgeDir()
	if err != nil {
		return err
	}

	socketPath := filepath.Join(forgeDir, "forge.sock")
	pidFile := filepath.Join(forgeDir, "forge.pid")

	// Check socket
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		fmt.Println("⭘ Daemon is not running")
		return nil
	}

	// Read PID
	data, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Println("⚠ Socket exists but no PID file (stale socket?)")
		return nil
	}

	var pid int
	_, _ = fmt.Sscanf(string(data), "%d", &pid)

	fmt.Printf("● Daemon is running\n")
	fmt.Printf("  PID: %d\n", pid)
	fmt.Printf("  Socket: %s\n", socketPath)

	// TODO: Connect to daemon and get detailed status
	return nil
}

