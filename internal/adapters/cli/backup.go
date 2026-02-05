// Package cli implements the command-line interface for Forge.
package cli

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/forge-platform/forge/internal/adapters/daemon"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup and restore operations",
	Long:  "Create backups of the Forge database and restore from backups.",
}

var backupCreateCmd = &cobra.Command{
	Use:   "create [output-file]",
	Short: "Create a backup of the Forge database",
	Long: `Create a backup of the Forge database.
If no output file is specified, a timestamped file will be created in the current directory.`,
	RunE: runBackupCreate,
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-file>",
	Short: "Restore from a backup",
	Long:  "Restore the Forge database from a backup file.",
	Args:  cobra.ExactArgs(1),
	RunE:  runBackupRestore,
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	RunE:  runBackupList,
}

var (
	backupCompress bool
	backupForce    bool
)

func init() {
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupListCmd)

	backupCreateCmd.Flags().BoolVar(&backupCompress, "compress", true, "Compress backup with gzip")
	backupRestoreCmd.Flags().BoolVarP(&backupForce, "force", "f", false, "Force restore without confirmation")
}

func runBackupCreate(cmd *cobra.Command, args []string) error {
	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	// Get database path from daemon
	resp, err := client.Call(context.Background(), "backup.info", nil)
	if err != nil {
		return fmt.Errorf("failed to get backup info: %w", err)
	}

	dbPath, _ := resp["db_path"].(string)
	if dbPath == "" {
		return fmt.Errorf("failed to get database path")
	}

	// Determine output file
	var outputFile string
	if len(args) > 0 {
		outputFile = args[0]
	} else {
		timestamp := time.Now().Format("20060102-150405")
		outputFile = fmt.Sprintf("forge-backup-%s.db", timestamp)
		if backupCompress {
			outputFile += ".gz"
		}
	}

	// Copy database file
	fmt.Printf("Creating backup from %s to %s...\n", dbPath, outputFile)

	srcFile, err := os.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dstFile.Close()

	var writer io.Writer = dstFile
	if backupCompress {
		gzWriter := gzip.NewWriter(dstFile)
		defer gzWriter.Close()
		writer = gzWriter
	}

	bytesWritten, err := io.Copy(writer, srcFile)
	if err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	info, _ := os.Stat(outputFile)
	fmt.Printf("Backup created successfully!\n")
	fmt.Printf("  Source size:  %.2f MB\n", float64(bytesWritten)/1024/1024)
	if info != nil {
		fmt.Printf("  Backup size:  %.2f MB\n", float64(info.Size())/1024/1024)
		if backupCompress {
			fmt.Printf("  Compression:  %.1f%%\n", (1-float64(info.Size())/float64(bytesWritten))*100)
		}
	}
	fmt.Printf("  Output file:  %s\n", outputFile)

	return nil
}

func runBackupRestore(cmd *cobra.Command, args []string) error {
	backupFile := args[0]

	// Check if backup file exists
	info, err := os.Stat(backupFile)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	fmt.Printf("Backup file: %s (%.2f MB)\n", backupFile, float64(info.Size())/1024/1024)

	if !backupForce {
		fmt.Print("This will overwrite the current database. Continue? [y/N]: ")
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Restore cancelled.")
			return nil
		}
	}

	// Connect to daemon to get database path and stop it
	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Call(context.Background(), "backup.info", nil)
	if err != nil {
		return fmt.Errorf("failed to get backup info: %w", err)
	}

	dbPath, _ := resp["db_path"].(string)
	if dbPath == "" {
		return fmt.Errorf("failed to get database path")
	}

	fmt.Println("Stopping daemon for restore...")
	_, _ = client.Call(context.Background(), "shutdown", nil)
	client.Close()

	// Wait for daemon to stop
	time.Sleep(2 * time.Second)

	// Open backup file
	srcFile, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer srcFile.Close()

	var reader io.Reader = srcFile
	if filepath.Ext(backupFile) == ".gz" {
		gzReader, err := gzip.NewReader(srcFile)
		if err != nil {
			return fmt.Errorf("failed to decompress backup: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Create new database file
	dstFile, err := os.Create(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database file: %w", err)
	}
	defer dstFile.Close()

	bytesWritten, err := io.Copy(dstFile, reader)
	if err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	fmt.Printf("Database restored successfully!\n")
	fmt.Printf("  Restored size: %.2f MB\n", float64(bytesWritten)/1024/1024)
	fmt.Println("Please restart the daemon with: forge start")

	return nil
}

func runBackupList(cmd *cobra.Command, args []string) error {
	// List backup files in current directory
	matches, err := filepath.Glob("forge-backup-*.db*")
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		fmt.Println("No backup files found in current directory.")
		return nil
	}

	fmt.Println("Available backups:")
	for _, f := range matches {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		fmt.Printf("  %s\t%.2f MB\t%s\n", f, float64(info.Size())/1024/1024, info.ModTime().Format("2006-01-02 15:04:05"))
	}

	return nil
}

