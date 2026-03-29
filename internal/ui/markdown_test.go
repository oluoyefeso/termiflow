package ui

import (
	"strings"
	"testing"
)

func TestRenderMarkdownHeading(t *testing.T) {
	out := RenderMarkdown("# Hello World", 72)
	// glamour adds ANSI styling, so output should be longer than raw input
	if len(out) <= len("# Hello World") {
		t.Errorf("expected styled output longer than raw input, got %d bytes", len(out))
	}
	// The heading text should still be present (without the # prefix)
	if !strings.Contains(out, "Hello World") {
		t.Errorf("expected output to contain 'Hello World', got %q", out)
	}
}

func TestRenderMarkdownEmpty(t *testing.T) {
	out := RenderMarkdown("", 72)
	// Should not panic, may return empty or whitespace
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty-ish output for empty input, got %q", out)
	}
}

func TestRenderMarkdownZeroWidth(t *testing.T) {
	// Should not panic with zero or negative width
	out := RenderMarkdown("test content", 0)
	if !strings.Contains(out, "test content") {
		t.Errorf("expected output to contain 'test content', got %q", out)
	}

	out = RenderMarkdown("test content", -1)
	if !strings.Contains(out, "test content") {
		t.Errorf("expected output to contain 'test content' with negative width, got %q", out)
	}
}

func TestRenderMarkdownCodeBlock(t *testing.T) {
	input := "```go\nfunc main() {}\n```"
	out := RenderMarkdown(input, 72)
	if !strings.Contains(out, "func main()") {
		t.Errorf("expected code block content preserved, got %q", out)
	}
}

func TestRenderMarkdownFallback(t *testing.T) {
	// Normal markdown should render successfully (not trigger fallback)
	input := "**bold** and *italic*"
	out := RenderMarkdown(input, 72)
	// Should contain the text content regardless of styling
	if !strings.Contains(out, "bold") || !strings.Contains(out, "italic") {
		t.Errorf("expected text content preserved, got %q", out)
	}
}

func TestGlamourStyleCaching(t *testing.T) {
	if err := LoadTheme("amber"); err != nil {
		t.Fatalf("LoadTheme error: %v", err)
	}
	if GetGlamourStyle() == nil {
		t.Error("expected cachedGlamourStyle to be non-nil after LoadTheme")
	}
}

func TestGlamourThemed(t *testing.T) {
	if err := LoadTheme("dracula"); err != nil {
		t.Fatalf("LoadTheme error: %v", err)
	}
	out := RenderMarkdown("# Hello", 72)
	if !strings.Contains(out, "Hello") {
		t.Errorf("themed render should contain 'Hello', got %q", out)
	}
	// Output should be styled (longer than raw input)
	if len(out) <= len("# Hello") {
		t.Errorf("expected styled output longer than raw input, got %d bytes", len(out))
	}
}
