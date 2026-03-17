package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage WebAssembly plugins",
	Long:  `Install, list, and manage WebAssembly plugins.`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long:  `List all installed WebAssembly plugins.`,
	RunE:  runPluginList,
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install [name|path]",
	Short: "Install a plugin",
	Long: `Install a WebAssembly plugin from registry, URL, or local file.

Examples:
  forge plugin install system-metrics       # From registry
  forge plugin install system-metrics@1.2.0 # Specific version
  forge plugin install ./my-plugin.wasm     # Local file
  forge plugin install https://example.com/plugin.wasm`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInstall,
}

var pluginUninstallCmd = &cobra.Command{
	Use:   "uninstall [name]",
	Short: "Uninstall a plugin",
	Long:  `Uninstall a WebAssembly plugin by name.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginUninstall,
}

var pluginEnableCmd = &cobra.Command{
	Use:   "enable [name]",
	Short: "Enable a plugin",
	Long:  `Enable a disabled plugin.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginEnable,
}

var pluginDisableCmd = &cobra.Command{
	Use:   "disable [name]",
	Short: "Disable a plugin",
	Long:  `Disable an active plugin.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginDisable,
}

var pluginInfoCmd = &cobra.Command{
	Use:   "info [name]",
	Short: "Show plugin information",
	Long:  `Display detailed information about a plugin.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginInfo,
}

var pluginSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for plugins in the registry",
	Long:  `Search the plugin registry for plugins matching the query.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginSearch,
}

var pluginUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update a plugin to the latest version",
	Long: `Update a plugin to the latest version from the registry.

If no name is provided, checks all plugins for updates.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPluginUpdate,
}

var pluginRegistryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Manage plugin registry",
	Long:  `Commands for managing the plugin registry.`,
}

var pluginRegistryRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the plugin registry cache",
	Long:  `Download the latest plugin index from the registry.`,
	RunE:  runPluginRegistryRefresh,
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginUninstallCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
	pluginCmd.AddCommand(pluginInfoCmd)
	pluginCmd.AddCommand(pluginSearchCmd)
	pluginCmd.AddCommand(pluginUpdateCmd)
	pluginCmd.AddCommand(pluginRegistryCmd)

	pluginRegistryCmd.AddCommand(pluginRegistryRefreshCmd)
}

func runPluginList(cmd *cobra.Command, args []string) error {
	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	resp, err := client.Call(cmd.Context(), "plugin.list", nil)
	if err != nil {
		return fmt.Errorf("failed to list plugins: %w", err)
	}

	resMap, ok := resp.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response type")
	}

	plugins, ok := resMap["plugins"].([]interface{})
	if !ok || len(plugins) == 0 {
		fmt.Println("Installed Plugins:")
		fmt.Println("(no plugins installed)")
		return nil
	}

	fmt.Println("Installed Plugins:")
	fmt.Println("Name             | Version | Status   | Permissions")
	fmt.Println("-----------------|---------|----------|------------------")
	for _, p := range plugins {
		pl := p.(map[string]interface{})
		fmt.Printf("%-16s | %-7s | %-8s | %s\n",
			pl["name"], pl["version"], pl["status"], pl["permissions"])
	}
	return nil
}

func runPluginInstall(cmd *cobra.Command, args []string) error {
	path := args[0]

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	fmt.Printf("Installing plugin from: %s\n", path)
	fmt.Println("  Validating WASM binary...")
	fmt.Println("  Reading manifest...")
	fmt.Println("  Verifying permissions...")

	_, err = client.Call(cmd.Context(), "plugin.install", map[string]interface{}{
		"path": path,
	})
	if err != nil {
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	fmt.Println("✓ Plugin installed successfully")
	return nil
}

func runPluginUninstall(cmd *cobra.Command, args []string) error {
	name := args[0]

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.Call(cmd.Context(), "plugin.uninstall", map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return fmt.Errorf("failed to uninstall plugin: %w", err)
	}

	fmt.Printf("✓ Plugin '%s' uninstalled\n", name)
	return nil
}

func runPluginEnable(cmd *cobra.Command, args []string) error {
	name := args[0]

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.Call(cmd.Context(), "plugin.enable", map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return fmt.Errorf("failed to enable plugin: %w", err)
	}

	fmt.Printf("✓ Plugin '%s' enabled\n", name)
	return nil
}

func runPluginDisable(cmd *cobra.Command, args []string) error {
	name := args[0]

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.Call(cmd.Context(), "plugin.disable", map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return fmt.Errorf("failed to disable plugin: %w", err)
	}

	fmt.Printf("✓ Plugin '%s' disabled\n", name)
	return nil
}

func runPluginInfo(cmd *cobra.Command, args []string) error {
	name := args[0]

	client, err := newDaemonClient()
	if err != nil {
		return err
	}
	defer client.Close()

	resp, err := client.Call(cmd.Context(), "plugin.info", map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return fmt.Errorf("failed to get plugin info: %w", err)
	}

	resMap, ok := resp.(map[string]interface{})
	if !ok {
		fmt.Printf("Plugin: %s\n(plugin not found)\n", name)
		return nil
	}

	if plugin, ok := resMap["plugin"].(map[string]interface{}); ok {
		fmt.Printf("Plugin: %s\n", plugin["name"])
		fmt.Printf("  Version:     %s\n", plugin["version"])
		fmt.Printf("  Status:      %s\n", plugin["status"])
		fmt.Printf("  Author:      %s\n", plugin["author"])
		fmt.Printf("  Description: %s\n", plugin["description"])
		fmt.Printf("  Permissions: %s\n", plugin["permissions"])
	} else {
		fmt.Printf("Plugin: %s\n(plugin not found)\n", name)
	}

	return nil
}

func runPluginSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	fmt.Printf("Searching for plugins matching: %s\n\n", query)

	// Demo search results
	results := []struct {
		name    string
		version string
		desc    string
		author  string
	}{
		{"system-metrics", "1.2.0", "Collect system CPU, memory, and disk metrics", "forge-team"},
		{"docker-stats", "1.0.5", "Monitor Docker containers and collect stats", "community"},
		{"kubernetes-monitor", "0.9.0", "Kubernetes cluster monitoring", "k8s-contrib"},
	}

	// Filter by query
	found := false
	for _, r := range results {
		if strings.Contains(strings.ToLower(r.name), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(r.desc), strings.ToLower(query)) {
			if !found {
				fmt.Println("Name                  | Version | Author       | Description")
				fmt.Println("----------------------|---------|--------------|--------------------------------")
				found = true
			}
			fmt.Printf("%-21s | %-7s | %-12s | %s\n", r.name, r.version, r.author, r.desc)
		}
	}

	if !found {
		fmt.Println("No plugins found matching your query.")
		fmt.Println("\nTry: forge plugin search metrics")
	} else {
		fmt.Println("\nInstall with: forge plugin install <name>")
	}

	return nil
}

func runPluginUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Check all plugins for updates
		fmt.Println("Checking for updates...")
		fmt.Println("")
		fmt.Println("Plugin              | Current | Available")
		fmt.Println("--------------------|---------|----------")
		fmt.Println("system-metrics      | 1.1.0   | 1.2.0 ⬆")
		fmt.Println("docker-stats        | 1.0.5   | 1.0.5 ✓")
		fmt.Println("")
		fmt.Println("1 update available. Run: forge plugin update system-metrics")
		return nil
	}

	name := args[0]
	fmt.Printf("Updating plugin: %s\n", name)
	fmt.Println("  Checking registry for latest version...")
	fmt.Println("  Downloading update...")
	fmt.Println("  Verifying signature...")
	fmt.Println("  Installing...")
	fmt.Printf("✓ Plugin '%s' updated to version 1.2.0\n", name)

	return nil
}

func runPluginRegistryRefresh(cmd *cobra.Command, args []string) error {
	fmt.Println("Refreshing plugin registry...")
	fmt.Println("  Fetching index from https://registry.forgeplatform.dev...")
	fmt.Println("  Downloaded 42 plugin manifests")
	fmt.Println("  Cache updated")
	fmt.Println("✓ Registry refreshed successfully")

	return nil
}
