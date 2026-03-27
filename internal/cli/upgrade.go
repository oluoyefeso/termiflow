package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/notifications"
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
		sp.Stop()
		// Try cache fallback
		if upgradeFromCache() {
			return nil
		}
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
		sp.Stop()
		// Try cache fallback
		if upgradeFromCache() {
			return nil
		}
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
	if !isNewerVersion(latest, current) {
		sp.Success(fmt.Sprintf("You're on the latest version (%s)", release.TagName))
		return nil
	}

	showUpgradeInfo(current, release.TagName, release.HTMLURL)
	return nil
}

func showUpgradeInfo(current, tagName, releaseURL string) {
	fmt.Println()
	fmt.Println(ui.Header("termiflow upgrade"))
	fmt.Println()
	fmt.Printf("  Current version: %s\n", ui.MutedStyle.Render("v"+current))
	fmt.Printf("  Latest version:  %s\n", ui.TitleStyle.Render(tagName))
	fmt.Println()
	fmt.Println(ui.BoldStyle.Render("  Upgrade with:"))
	fmt.Printf("    %s\n", ui.TitleStyle.Render(
		fmt.Sprintf("go install github.com/oluoyefeso/termiflow/cmd/termiflow@%s", tagName)))
	fmt.Println()
	if releaseURL != "" {
		fmt.Printf("  Release notes: %s\n", ui.MutedStyle.Render(releaseURL))
		fmt.Println()
	}
}

// upgradeFromCache attempts to show upgrade info from the notification cache.
// Returns true if a newer version was found in the cache.
func upgradeFromCache() bool {
	nm := notifications.NewManager(
		config.Get().Providers.Managed.BaseURL,
		config.Get().Providers.Managed.APIKey,
		version,
		config.GetCacheDir(),
		config.IsManagedMode(),
	)
	nm.LoadCache()

	latestVersion := nm.GetLatestVersion()
	if latestVersion == "" {
		return false
	}

	current := strings.TrimPrefix(version, "v")
	if !isNewerVersion(latestVersion, current) {
		return false
	}

	tagName := "v" + latestVersion
	showUpgradeInfo(current, tagName, "")
	return true
}

// isNewerVersion returns true if candidate is newer than current using semver.
func isNewerVersion(candidate, current string) bool {
	candidateVer, err := semver.NewVersion(candidate)
	if err != nil {
		return false
	}
	currentVer, err := semver.NewVersion(current)
	if err != nil {
		return false
	}
	return candidateVer.GreaterThan(currentVer)
}
