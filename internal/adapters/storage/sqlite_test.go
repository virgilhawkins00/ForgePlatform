// Package storage implements the SQLite-based persistence layer.
package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("/data")

	if cfg.Path != "/data/forge.db" {
		t.Errorf("expected path '/data/forge.db', got '%s'", cfg.Path)
	}
	if cfg.JournalMode != "WAL" {
		t.Errorf("expected journal mode 'WAL', got '%s'", cfg.JournalMode)
	}
	if cfg.Synchronous != "NORMAL" {
		t.Errorf("expected synchronous 'NORMAL', got '%s'", cfg.Synchronous)
	}
	if cfg.CacheSize != -64000 {
		t.Errorf("expected cache size -64000, got %d", cfg.CacheSize)
	}
	if cfg.MmapSize != 268435456 {
		t.Errorf("expected mmap size 268435456, got %d", cfg.MmapSize)
	}
	if cfg.BusyTimeout != 5000 {
		t.Errorf("expected busy timeout 5000, got %d", cfg.BusyTimeout)
	}
}

func TestNew(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "forge-sqlite-test")
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig(tmpDir)

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("expected non-nil db")
	}
	if db.conn == nil {
		t.Error("expected non-nil connection")
	}
}

func TestDB_Conn(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "forge-sqlite-test-conn")
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig(tmpDir)
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer db.Close()

	conn := db.Conn()
	if conn == nil {
		t.Error("expected non-nil sql.DB from Conn()")
	}
}

func TestDB_Close(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "forge-sqlite-test-close")
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig(tmpDir)
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestDB_Ping(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "forge-sqlite-test-ping")
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig(tmpDir)
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer db.Close()

	err = db.Ping(context.Background())
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestDB_Path(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "forge-sqlite-test-path")
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig(tmpDir)
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer db.Close()

	path := db.Path()
	if path != cfg.Path {
		t.Errorf("expected path '%s', got '%s'", cfg.Path, path)
	}
}

func TestDB_BeginTx(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "forge-sqlite-test-tx")
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig(tmpDir)
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer db.Close()

	tx, err := db.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}
	if tx == nil {
		t.Error("expected non-nil transaction")
	}
	_ = tx.Rollback()
}

func TestNewTaskRepository(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "forge-sqlite-test-task-repo")
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig(tmpDir)
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer db.Close()

	repo := NewTaskRepository(db)
	if repo == nil {
		t.Error("expected non-nil task repository")
	}
}

func TestNewMetricRepository(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "forge-sqlite-test-metric-repo")
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig(tmpDir)
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer db.Close()

	repo := NewMetricRepository(db)
	if repo == nil {
		t.Error("expected non-nil metric repository")
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		Path:        "/custom/path/db.sqlite",
		JournalMode: "DELETE",
		Synchronous: "FULL",
		CacheSize:   -32000,
		MmapSize:    134217728,
		BusyTimeout: 10000,
	}

	if cfg.Path != "/custom/path/db.sqlite" {
		t.Error("Path field mismatch")
	}
	if cfg.JournalMode != "DELETE" {
		t.Error("JournalMode field mismatch")
	}
	if cfg.Synchronous != "FULL" {
		t.Error("Synchronous field mismatch")
	}
	if cfg.CacheSize != -32000 {
		t.Error("CacheSize field mismatch")
	}
}

