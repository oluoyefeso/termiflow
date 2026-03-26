package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/ui"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Check for newer versions and show upgrade instructions",
	RunE:  runUpgrade,
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	sp := ui.NewSpinner("Checking for updates...")
	sp.Start()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/oluoyefeso/termiflow/releases/latest")
	if err != nil {
		sp.Error("Failed to check for updates")
		return fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		sp.Stop()
		fmt.Println()
		fmt.Print(ui.Warning("No releases published yet"))
		fmt.Printf("  You're running version: %s\n", ui.MutedStyle.Render(version))
		fmt.Println()
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		sp.Error("Failed to check for updates")
		return fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		sp.Error("Failed to parse release info")
		return err
	}

	current := strings.TrimPrefix(version, "v")
	if current == "dev" || current == "" {
		sp.Stop()
		fmt.Println()
		fmt.Print(ui.Warning("Running dev build — cannot compare versions"))
		fmt.Printf("  Latest release: %s\n", ui.TitleStyle.Render(release.TagName))
		fmt.Printf("  %s\n", release.HTMLURL)
		return nil
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	if current >= latest {
		sp.Success(fmt.Sprintf("You're on the latest version (%s)", release.TagName))
		return nil
	}

	sp.Stop()
	fmt.Println()
	fmt.Println(ui.Header("termiflow upgrade"))
	fmt.Println()
	fmt.Printf("  Current version: %s\n", ui.MutedStyle.Render("v"+current))
	fmt.Printf("  Latest version:  %s\n", ui.TitleStyle.Render(release.TagName))
	fmt.Println()
	fmt.Println(ui.BoldStyle.Render("  Upgrade with:"))
	fmt.Printf("    %s\n", ui.TitleStyle.Render(
		fmt.Sprintf("go install github.com/oluoyefeso/termiflow/cmd/termiflow@%s", release.TagName)))
	fmt.Println()
	fmt.Printf("  Release notes: %s\n", ui.MutedStyle.Render(release.HTMLURL))
	fmt.Println()

	return nil
}
