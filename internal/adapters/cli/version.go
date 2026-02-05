package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build information, set via ldflags during build.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Display the version, commit hash, and build date of Forge.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Forge Platform\n")
		fmt.Printf("  Version:    %s\n", Version)
		fmt.Printf("  Commit:     %s\n", Commit)
		fmt.Printf("  Built:      %s\n", BuildDate)
		fmt.Printf("  Go version: %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

