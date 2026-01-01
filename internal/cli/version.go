package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/termiflow/termiflow/internal/ui"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and build information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("termiflow %s\n\n", version)
		fmt.Print(ui.Info("Build", date))
		fmt.Print(ui.Info("Commit", commit))
		fmt.Print(ui.Info("Go", runtime.Version()))
		fmt.Print(ui.Info("Platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)))
	},
}
