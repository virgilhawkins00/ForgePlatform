// Package storage implements the SQLite-based persistence layer.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Config holds SQLite configuration options.
type Config struct {
	Path       string
	JournalMode string // WAL, DELETE, TRUNCATE
	Synchronous string // OFF, NORMAL, FULL
	CacheSize   int    // in KB (negative for KB, positive for pages)
	MmapSize    int64  // in bytes
	BusyTimeout int    // in milliseconds
}

// DefaultConfig returns the default SQLite configuration optimized for TSDB.
func DefaultConfig(dataDir string) Config {
	return Config{
		Path:        filepath.Join(dataDir, "forge.db"),
		JournalMode: "WAL",
		Synchronous: "NORMAL",
		CacheSize:   -64000, // 64MB
		MmapSize:    268435456, // 256MB
		BusyTimeout: 5000,
	}
}

// DB wraps the SQLite database connection.
type DB struct {
	conn   *sql.DB
	config Config
}

// New creates a new SQLite database connection with TSDB optimizations.
func New(config Config) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database with optimized settings
	dsn := fmt.Sprintf("%s?_journal_mode=%s&_synchronous=%s&_busy_timeout=%d",
		config.Path,
		config.JournalMode,
		config.Synchronous,
		config.BusyTimeout,
	)

	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{
		conn:   conn,
		config: config,
	}

	// Apply additional PRAGMAs
	if err := db.applyPragmas(); err != nil {
		conn.Close()
		return nil, err
	}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

// applyPragmas applies SQLite performance optimizations.
func (db *DB) applyPragmas() error {
	pragmas := []string{
		fmt.Sprintf("PRAGMA cache_size = %d", db.config.CacheSize),
		fmt.Sprintf("PRAGMA mmap_size = %d", db.config.MmapSize),
		"PRAGMA temp_store = MEMORY",
		"PRAGMA foreign_keys = ON",
	}

	for _, pragma := range pragmas {
		if _, err := db.conn.Exec(pragma); err != nil {
			return fmt.Errorf("failed to apply pragma %q: %w", pragma, err)
		}
	}

	return nil
}

// initSchema creates the database tables if they don't exist.
func (db *DB) initSchema() error {
	schema := `
	-- Metrics table (TSDB)
	CREATE TABLE IF NOT EXISTS metrics (
		id BLOB(16) PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		value REAL NOT NULL,
		timestamp INTEGER NOT NULL,
		series_hash INTEGER NOT NULL,
		tags JSON
	);
	CREATE INDEX IF NOT EXISTS idx_metrics_series_time ON metrics(series_hash, timestamp);
	CREATE INDEX IF NOT EXISTS idx_metrics_name_time ON metrics(name, timestamp);

	-- Aggregated metrics for downsampling
	CREATE TABLE IF NOT EXISTS metrics_aggregated (
		id BLOB(16) PRIMARY KEY,
		name TEXT NOT NULL,
		series_hash INTEGER NOT NULL,
		window_start INTEGER NOT NULL,
		window_end INTEGER NOT NULL,
		resolution TEXT NOT NULL,
		count INTEGER NOT NULL,
		sum REAL NOT NULL,
		min REAL NOT NULL,
		max REAL NOT NULL,
		avg REAL NOT NULL,
		tags JSON
	);
	CREATE INDEX IF NOT EXISTS idx_metrics_agg_series ON metrics_aggregated(series_hash, resolution, window_start);

	-- Tasks table (Durable Queue)
	CREATE TABLE IF NOT EXISTS tasks (
		id BLOB(16) PRIMARY KEY,
		type TEXT NOT NULL,
		payload JSON,
		status TEXT DEFAULT 'PENDING',
		priority INTEGER DEFAULT 0,
		max_retries INTEGER DEFAULT 3,
		retry_count INTEGER DEFAULT 0,
		run_at INTEGER NOT NULL,
		locked_until INTEGER,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		completed_at INTEGER,
		error TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_poll ON tasks(status, run_at) WHERE status = 'PENDING';
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

	-- Plugins table
	CREATE TABLE IF NOT EXISTS plugins (
		id BLOB(16) PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		version TEXT NOT NULL,
		description TEXT,
		author TEXT,
		path TEXT NOT NULL,
		hash TEXT NOT NULL,
		status TEXT DEFAULT 'inactive',
		permissions JSON,
		config JSON,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		loaded_at INTEGER,
		error TEXT
	);

	-- Conversations table (AI)
	CREATE TABLE IF NOT EXISTS conversations (
		id BLOB(16) PRIMARY KEY,
		title TEXT NOT NULL,
		model TEXT NOT NULL,
		messages JSON NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at DESC);

	-- Workflows table
	CREATE TABLE IF NOT EXISTS workflows (
		id BLOB(16) PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		steps JSON NOT NULL,
		variables JSON,
		status TEXT DEFAULT 'pending',
		current_step INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		started_at INTEGER,
		completed_at INTEGER,
		error TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows(status);
	`

	_, err := db.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying database connection.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// BeginTx starts a new transaction.
func (db *DB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return db.conn.BeginTx(ctx, nil)
}

