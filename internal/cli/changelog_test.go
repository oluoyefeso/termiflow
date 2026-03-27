package cli

import (
	"strings"
	"testing"
)

func TestChangelogCmd(t *testing.T) {
	if changelogCmd == nil {
		t.Fatal("changelogCmd should not be nil")
	}
	if changelogCmd.Use != "changelog" {
		t.Errorf("changelogCmd.Use = %q, want %q", changelogCmd.Use, "changelog")
	}
	if changelogCmd.Flags().Lookup("all") == nil {
		t.Error("changelog command missing --all flag")
	}
}

func TestColorReleaseTags(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"**New:** Added feature X", "New:"},
		{"**Fix:** Fixed bug Y", "Fix:"},
		{"**Breaking:** Removed Z", "Breaking:"},
		{"No tags here", "No tags here"},
		{"**Added:** Another feature", "Added:"},
		{"**Fixed:** Another bug", "Fixed:"},
	}

	for _, tt := range tests {
		result := colorReleaseTags(tt.input)
		// The result should contain the plain tag text (without ** markers)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("colorReleaseTags(%q) should contain %q, got %q", tt.input, tt.contains, result)
		}
		// Should NOT contain the original markdown bold markers
		if strings.Contains(result, "**") {
			// If the tag was recognized, bold markers should be replaced
			if strings.Contains(tt.input, "**New:**") ||
				strings.Contains(tt.input, "**Fix:**") ||
				strings.Contains(tt.input, "**Breaking:**") ||
				strings.Contains(tt.input, "**Added:**") ||
				strings.Contains(tt.input, "**Fixed:**") {
				t.Errorf("colorReleaseTags(%q) should not contain ** markers, got %q", tt.input, result)
			}
		}
	}
}

func TestRenderReleaseBody_EmptyBody(t *testing.T) {
	// Should not panic on empty body
	renderReleaseBody("")
}

func TestRenderReleaseBody_SectionHeaders(t *testing.T) {
	// Should not panic on various markdown structures
	body := `## Added
- Feature A
- Feature B

### Fixed
- Bug C

Regular text line`
	renderReleaseBody(body)
}

func TestChangelogOutputJSON_EmptyReleases(t *testing.T) {
	out := ChangelogOutputJSON{
		Releases: []ChangelogReleaseJSON{},
		Current:  "v1.0.0",
	}
	if len(out.Releases) != 0 {
		t.Error("empty releases should have length 0")
	}
	if out.Current != "v1.0.0" {
		t.Errorf("current = %q, want %q", out.Current, "v1.0.0")
	}
}
