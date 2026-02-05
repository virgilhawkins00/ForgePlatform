// Package daemon implements the background daemon service.
package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/adapters/storage"
	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/forge-platform/forge/internal/core/services"
)

// Server represents the Forge daemon server.
type Server struct {
	config      Config
	listener    net.Listener
	db          *storage.DB
	logger      ports.Logger
	taskSvc     *services.TaskService
	metricSvc   *services.MetricService
	ragSvc      *services.RAGService
	workflowSvc *services.WorkflowService
	alertSvc    *services.AlertService
	aiProvider  ports.AIProvider
	startedAt   time.Time
	stopCh      chan struct{}
	wg          sync.WaitGroup
	mu          sync.RWMutex
	running     bool
}

// Config holds daemon configuration.
type Config struct {
	SocketPath      string
	PIDFile         string
	DataDir         string
	ShutdownTimeout time.Duration
	WorkerCount     int
}

// DefaultConfig returns the default daemon configuration.
func DefaultConfig(forgeDir string) Config {
	return Config{
		SocketPath:      filepath.Join(forgeDir, "forge.sock"),
		PIDFile:         filepath.Join(forgeDir, "forge.pid"),
		DataDir:         filepath.Join(forgeDir, "data"),
		ShutdownTimeout: 10 * time.Second,
		WorkerCount:     4,
	}
}

// NewServer creates a new daemon server.
func NewServer(config Config, logger ports.Logger) (*Server, error) {
	// Initialize database
	dbConfig := storage.DefaultConfig(config.DataDir)
	db, err := storage.New(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize repositories
	taskRepo := storage.NewTaskRepository(db)
	metricRepo := storage.NewMetricRepository(db)

	// Initialize services
	taskSvc := services.NewTaskService(taskRepo, logger)
	metricSvc := services.NewMetricService(metricRepo, logger, services.DefaultMetricServiceConfig())
	ragSvc := services.NewRAGService(metricRepo, taskRepo, logger, services.RAGConfig{})
	workflowSvc := services.NewWorkflowService(nil, nil, logger)

	// Register built-in workflow actions
	workflowSvc.RegisterAction(domain.StepTypeShell, services.NewShellAction(""))
	workflowSvc.RegisterAction(domain.StepTypeHTTP, services.NewHTTPAction(30*time.Second))
	workflowSvc.RegisterAction(domain.StepTypeMetric, services.NewMetricAction(metricRepo))
	workflowSvc.RegisterAction(domain.StepTypeTask, services.NewTaskAction(taskRepo))

	// Initialize alert service (with nil repos for now - can be enhanced later)
	alertSvc := services.NewAlertService(nil, nil, nil, nil, metricRepo, logger)

	return &Server{
		config:      config,
		db:          db,
		logger:      logger,
		taskSvc:     taskSvc,
		metricSvc:   metricSvc,
		ragSvc:      ragSvc,
		workflowSvc: workflowSvc,
		alertSvc:    alertSvc,
		stopCh:      make(chan struct{}),
	}, nil
}

// SetAIProvider sets the AI provider for the server.
func (s *Server) SetAIProvider(provider ports.AIProvider) {
	s.aiProvider = provider
}

// Start starts the daemon server.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("daemon already running")
	}
	s.running = true
	s.startedAt = time.Now()
	s.mu.Unlock()

	// Remove stale socket
	os.Remove(s.config.SocketPath)

	// Create Unix socket listener
	listener, err := net.Listen("unix", s.config.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	s.listener = listener

	// Write PID file
	if err := os.WriteFile(s.config.PIDFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		listener.Close()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	s.logger.Info("Daemon started", "socket", s.config.SocketPath, "pid", os.Getpid())

	// Start task workers
	s.taskSvc.StartWorkers(ctx, s.config.WorkerCount)

	// Start metric flusher
	s.metricSvc.Start(ctx, time.Second)

	// Start accepting connections
	s.wg.Add(1)
	go s.acceptConnections(ctx)

	// Start background downsampling job
	s.wg.Add(1)
	go s.runDownsamplingJob(ctx)

	return nil
}

// runDownsamplingJob runs periodic downsampling of old metrics.
// Retention policies from ForgePlatform.md:
// - Raw data: 7 days -> downsample to 1m
// - 1-minute aggregates: 30 days -> downsample to 1h
// - 1-hour aggregates: 1 year (no further downsampling)
func (s *Server) runDownsamplingJob(ctx context.Context) {
	defer s.wg.Done()

	// Run downsampling every hour
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	// Run once at startup after a short delay
	startupDelay := time.NewTimer(5 * time.Minute)
	defer startupDelay.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-startupDelay.C:
			s.runDownsampling(ctx)
		case <-ticker.C:
			s.runDownsampling(ctx)
		}
	}
}

// runDownsampling performs the actual downsampling operations.
func (s *Server) runDownsampling(ctx context.Context) {
	s.logger.Info("Starting scheduled downsampling...")

	// Downsample raw metrics older than 7 days to 1-minute resolution
	if err := s.metricSvc.Downsample(ctx, 7*24*time.Hour, "1m"); err != nil {
		s.logger.Error("Failed to downsample raw metrics", "error", err)
	}

	// Clean up old aggregated metrics based on retention policies
	if err := s.metricSvc.CleanupAggregated(ctx); err != nil {
		s.logger.Error("Failed to cleanup aggregated metrics", "error", err)
	}

	s.logger.Info("Scheduled downsampling completed")
}

// Stop gracefully stops the daemon.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	s.logger.Info("Stopping daemon...")

	// Signal stop
	close(s.stopCh)

	// Stop services
	s.taskSvc.StopWorkers()
	s.metricSvc.Stop(ctx)

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for goroutines
	s.wg.Wait()

	// Close database
	if s.db != nil {
		s.db.Close()
	}

	// Cleanup files
	os.Remove(s.config.SocketPath)
	os.Remove(s.config.PIDFile)

	s.logger.Info("Daemon stopped")
	return nil
}

// IsRunning returns whether the daemon is running.
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStatus returns the daemon status.
func (s *Server) GetStatus() ports.DaemonStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	uptime := ""
	if s.running {
		uptime = time.Since(s.startedAt).Round(time.Second).String()
	}

	return ports.DaemonStatus{
		Running:   s.running,
		StartedAt: s.startedAt.Format(time.RFC3339),
		Uptime:    uptime,
	}
}

