package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/ui"
)

var changelogAll bool

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Show release notes from recent versions",
	Long: `Show release notes from recent versions.

Examples:
  termiflow changelog          # Show latest 3 releases
  termiflow changelog --all    # Show all releases`,
	RunE: runChangelog,
}

func init() {
	changelogCmd.Flags().BoolVar(&changelogAll, "all", false, "show all releases")
}

// JSON output types for changelog command.

type ChangelogOutputJSON struct {
	Releases []ChangelogReleaseJSON `json:"releases"`
	Current  string                 `json:"current_version"`
}

type ChangelogReleaseJSON struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	PublishedAt string `json:"published_at"`
	URL         string `json:"url"`
	Body        string `json:"body"`
}

func runChangelog(cmd *cobra.Command, args []string) error {
	limit := 3
	if changelogAll {
		limit = 30
	}

	if !jsonOutput {
		sp := ui.NewSpinner("Fetching release notes...")
		sp.Start()
		defer sp.Stop()
	}

	releases, err := fetchReleases(limit)
	if err != nil {
		if jsonOutput {
			return ui.WriteJSONError(nil, err.Error(), version)
		}
		return err
	}

	if len(releases) == 0 {
		if jsonOutput {
			return ui.WriteJSON(ChangelogOutputJSON{
				Releases: []ChangelogReleaseJSON{},
				Current:  version,
			}, version)
		}
		fmt.Println()
		fmt.Print(ui.Warning("No releases published yet"))
		fmt.Printf("  You're running version: %s\n", ui.MutedStyle.Render(version))
		fmt.Println()
		return nil
	}

	if jsonOutput {
		return renderChangelogJSON(releases)
	}

	renderChangelogTerminal(releases)
	return nil
}

func renderChangelogJSON(releases []githubRelease) error {
	jsonReleases := make([]ChangelogReleaseJSON, 0, len(releases))
	for _, r := range releases {
		jsonReleases = append(jsonReleases, ChangelogReleaseJSON{
			TagName:     r.TagName,
			Name:        r.Name,
			PublishedAt: r.PublishedAt,
			URL:         r.HTMLURL,
			Body:        r.Body,
		})
	}
	return ui.WriteJSON(ChangelogOutputJSON{
		Releases: jsonReleases,
		Current:  version,
	}, version)
}

func renderChangelogTerminal(releases []githubRelease) {
	fmt.Println(ui.Header("termiflow changelog"))
	fmt.Println()

	current := strings.TrimPrefix(version, "v")

	for i, release := range releases {
		tag := release.TagName
		name := release.Name
		if name == "" {
			name = tag
		}

		// Mark current version
		marker := ""
		releaseVer := strings.TrimPrefix(tag, "v")
		if releaseVer == current {
			marker = ui.SuccessStyle.Render(" ← you are here")
		}

		fmt.Printf("  %s%s\n", ui.AccentStyle.Render(name), marker)

		// Published date
		if release.PublishedAt != "" {
			if t, err := time.Parse(time.RFC3339, release.PublishedAt); err == nil {
				fmt.Printf("  %s\n", ui.MutedStyle.Render(t.Format("02 Jan 2006")))
			}
		}

		// Body with tag coloring
		if release.Body != "" {
			fmt.Println()
			renderReleaseBody(release.Body)
		}

		if i < len(releases)-1 {
			fmt.Println()
			fmt.Print(ui.Divider())
		}
	}

	fmt.Println()
	fmt.Print(ui.Footer(len(releases), 0, ""))
}

func renderReleaseBody(body string) {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Section headers
		if strings.HasPrefix(trimmed, "### ") {
			header := strings.TrimPrefix(trimmed, "### ")
			fmt.Printf("  %s\n", ui.BoldStyle.Render(header))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			header := strings.TrimPrefix(trimmed, "## ")
			fmt.Printf("  %s\n", ui.AccentStyle.Render(header))
			continue
		}

		// Bullet items with tag coloring
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			content := trimmed[2:]
			content = colorReleaseTags(content)
			fmt.Printf("    %s %s\n", ui.MutedStyle.Render("›"), content)
			continue
		}

		// Skip empty lines but preserve spacing
		if trimmed == "" {
			continue
		}

		// Regular text
		fmt.Printf("    %s\n", ui.MutedStyle.Render(trimmed))
	}
}

func colorReleaseTags(line string) string {
	// Color common changelog tags
	replacements := []struct {
		tag   string
		style func(...string) string
	}{
		{"**New:**", ui.SuccessStyle.Render},
		{"**NEW:**", ui.SuccessStyle.Render},
		{"**Added:**", ui.SuccessStyle.Render},
		{"**ADDED:**", ui.SuccessStyle.Render},
		{"**Fix:**", ui.BlueStyle.Render},
		{"**FIX:**", ui.BlueStyle.Render},
		{"**Fixed:**", ui.BlueStyle.Render},
		{"**FIXED:**", ui.BlueStyle.Render},
		{"**Breaking:**", ui.ErrorStyle.Render},
		{"**BREAKING:**", ui.ErrorStyle.Render},
		{"**Changed:**", ui.WarningStyle.Render},
		{"**CHANGED:**", ui.WarningStyle.Render},
		{"**Removed:**", ui.ErrorStyle.Render},
		{"**REMOVED:**", ui.ErrorStyle.Render},
	}

	for _, r := range replacements {
		if strings.Contains(line, r.tag) {
			plain := strings.Trim(r.tag, "*")
			line = strings.Replace(line, r.tag, r.style(plain), 1)
		}
	}

	return line
}
