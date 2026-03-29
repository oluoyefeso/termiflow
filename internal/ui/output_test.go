package ui

import (
	"testing"
)

func TestWarmMutedItalicThemed(t *testing.T) {
	if err := LoadTheme("dracula"); err != nil {
		t.Fatalf("LoadTheme error: %v", err)
	}

	// After loading dracula, warmMutedItalic should have italic set
	if !warmMutedItalic.GetItalic() {
		t.Error("warmMutedItalic should be italic after LoadTheme")
	}
}
