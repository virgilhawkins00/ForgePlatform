package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/forge-platform/forge/internal/adapters/daemon"
	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long:  `Manage users, API keys, and sessions in the Forge platform.`,
}

var userCreateCmd = &cobra.Command{
	Use:   "create <username> <email>",
	Short: "Create a new user",
	Args:  cobra.ExactArgs(2),
	RunE:  runUserCreate,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE:  runUserList,
}

var userGetCmd = &cobra.Command{
	Use:   "get <username>",
	Short: "Get user details",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserGet,
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <username>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserDelete,
}

var userAPIKeyCmd = &cobra.Command{
	Use:   "apikey",
	Short: "API key management",
}

var userAPIKeyCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runAPIKeyCreate,
}

var userAPIKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	RunE:  runAPIKeyList,
}

var userAPIKeyRevokeCmd = &cobra.Command{
	Use:   "revoke <key-id>",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runAPIKeyRevoke,
}

var userAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit logs",
	RunE:  runAuditLogs,
}

var (
	userRole        string
	userPermissions []string
	auditLimit      int
	auditAction     string
)

func init() {
	userCreateCmd.Flags().StringVar(&userRole, "role", "viewer", "User role (admin, operator, viewer)")

	userAPIKeyCreateCmd.Flags().StringSliceVar(&userPermissions, "permissions", []string{"*"}, "API key permissions")

	userAuditCmd.Flags().IntVar(&auditLimit, "limit", 50, "Maximum number of entries")
	userAuditCmd.Flags().StringVar(&auditAction, "action", "", "Filter by action")

	userAPIKeyCmd.AddCommand(userAPIKeyCreateCmd, userAPIKeyListCmd, userAPIKeyRevokeCmd)
	userCmd.AddCommand(userCreateCmd, userListCmd, userGetCmd, userDeleteCmd, userAPIKeyCmd, userAuditCmd)
}

func runUserCreate(cmd *cobra.Command, args []string) error {
	username := args[0]
	email := args[1]

	// Read password securely
	fmt.Print("Enter password: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()

	fmt.Print("Confirm password: ")
	confirmBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()

	if string(passwordBytes) != string(confirmBytes) {
		return fmt.Errorf("passwords do not match")
	}

	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Call(context.Background(), "user.create", map[string]interface{}{
		"username": username,
		"email":    email,
		"password": string(passwordBytes),
		"role":     userRole,
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("✓ User created: %s (%s)\n", username, resp["id"])
	return nil
}

func runUserList(cmd *cobra.Command, args []string) error {
	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Call(context.Background(), "user.list", nil)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	users, _ := resp["users"].([]interface{})

	if len(users) == 0 {
		fmt.Println("No users found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tUSERNAME\tEMAIL\tROLE\tSTATUS\tLAST LOGIN")
	fmt.Fprintln(w, "--\t--------\t-----\t----\t------\t----------")

	for _, u := range users {
		user := u.(map[string]interface{})
		lastLogin := "Never"
		if ll, ok := user["last_login_at"].(string); ok && ll != "" {
			if t, err := time.Parse(time.RFC3339, ll); err == nil {
				lastLogin = t.Format("2006-01-02 15:04")
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			truncateID(getString(user, "id")),
			getString(user, "username"),
			getString(user, "email"),
			getString(user, "role"),
			getString(user, "status"),
			lastLogin,
		)
	}
	w.Flush()
	return nil
}

func runUserGet(cmd *cobra.Command, args []string) error {
	username := args[0]

	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Call(context.Background(), "user.get", map[string]interface{}{
		"username": username,
	})
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	data, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(data))
	return nil
}

func runUserDelete(cmd *cobra.Command, args []string) error {
	username := args[0]

	fmt.Printf("Are you sure you want to delete user '%s'? [y/N]: ", username)
	var confirm string
	_, _ = fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Cancelled")
		return nil
	}

	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	_, err = client.Call(context.Background(), "user.delete", map[string]interface{}{
		"username": username,
	})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	fmt.Printf("✓ User deleted: %s\n", username)
	return nil
}

func runAPIKeyCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Call(context.Background(), "apikey.create", map[string]interface{}{
		"name":        name,
		"permissions": userPermissions,
	})
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	fmt.Println("✓ API key created successfully!")
	fmt.Println()
	fmt.Printf("  Name:    %s\n", name)
	fmt.Printf("  ID:      %s\n", resp["id"])
	fmt.Printf("  Key:     %s\n", resp["key"])
	fmt.Println()
	fmt.Println("⚠️  Store this key securely - it will not be shown again!")
	return nil
}

func runAPIKeyList(cmd *cobra.Command, args []string) error {
	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	resp, err := client.Call(context.Background(), "apikey.list", nil)
	if err != nil {
		return fmt.Errorf("failed to list API keys: %w", err)
	}

	keys, _ := resp["keys"].([]interface{})
	if len(keys) == 0 {
		fmt.Println("No API keys found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tPREFIX\tPERMISSIONS\tEXPIRES\tLAST USED")
	fmt.Fprintln(w, "--\t----\t------\t-----------\t-------\t---------")

	for _, k := range keys {
		key := k.(map[string]interface{})
		expires := "Never"
		if exp, ok := key["expires_at"].(string); ok && exp != "" {
			if t, err := time.Parse(time.RFC3339, exp); err == nil {
				expires = t.Format("2006-01-02")
			}
		}
		lastUsed := "Never"
		if lu, ok := key["last_used_at"].(string); ok && lu != "" {
			if t, err := time.Parse(time.RFC3339, lu); err == nil {
				lastUsed = t.Format("2006-01-02 15:04")
			}
		}
		perms := "[]"
		if p, ok := key["permissions"].([]interface{}); ok {
			strs := make([]string, len(p))
			for i, v := range p {
				strs[i] = v.(string)
			}
			perms = strings.Join(strs, ",")
		}
		fmt.Fprintf(w, "%s\t%s\t%s...\t%s\t%s\t%s\n",
			truncateID(getString(key, "id")),
			getString(key, "name"),
			getString(key, "key_prefix"),
			perms,
			expires,
			lastUsed,
		)
	}
	w.Flush()
	return nil
}

func runAPIKeyRevoke(cmd *cobra.Command, args []string) error {
	keyID := args[0]

	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	_, err = client.Call(context.Background(), "apikey.revoke", map[string]interface{}{
		"id": keyID,
	})
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	fmt.Printf("✓ API key revoked: %s\n", keyID)
	return nil
}

func runAuditLogs(cmd *cobra.Command, args []string) error {
	client, err := daemon.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	params := map[string]interface{}{
		"limit": auditLimit,
	}
	if auditAction != "" {
		params["action"] = auditAction
	}

	resp, err := client.Call(context.Background(), "audit.list", params)
	if err != nil {
		return fmt.Errorf("failed to list audit logs: %w", err)
	}

	logs, _ := resp["logs"].([]interface{})
	if len(logs) == 0 {
		fmt.Println("No audit logs found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tACTION\tRESOURCE\tSUCCESS\tDETAILS")
	fmt.Fprintln(w, "---------\t------\t--------\t-------\t-------")

	for _, l := range logs {
		log := l.(map[string]interface{})
		ts := getString(log, "timestamp")
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			ts = t.Format("2006-01-02 15:04:05")
		}
		success := "✓"
		if s, ok := log["success"].(bool); ok && !s {
			success = "✗"
		}
		details := ""
		if errStr := getString(log, "error"); errStr != "" {
			details = errStr
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			ts,
			getString(log, "action"),
			getString(log, "resource"),
			success,
			details,
		)
	}
	w.Flush()
	return nil
}

func truncateID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// Ensure domain.UserRole is used (for import)
var _ = domain.RoleAdmin
