package scheduler

import (
	"testing"

	"github.com/oluoyefeso/termiflow/internal/providers/search"
)

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"simple tags", "<p>hello</p>", "hello"},
		{"nested tags", "<div><p>hello <b>world</b></p></div>", "hello world"},
		{"attributes", `<a href="http://example.com">link</a>`, "link"},
		{"whitespace collapse", "<p>hello</p>  \n  <p>world</p>", "hello world"},
		{"empty", "", ""},
		{"self-closing", "text<br/>more", "text more"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTML(tt.input)
			if got != tt.want {
				t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripHTMLUnicode(t *testing.T) {
	input := "<p>こんにちは <b>世界</b></p>"
	got := stripHTML(input)
	want := "こんにちは 世界"
	if got != want {
		t.Errorf("stripHTML(%q) = %q, want %q", input, got, want)
	}
}

func TestDeduplicateByURL(t *testing.T) {
	results := []search.SearchResult{
		{URL: "https://a.com/1", Title: "First"},
		{URL: "https://a.com/2", Title: "Second"},
		{URL: "https://a.com/1", Title: "Duplicate"},
	}

	deduped := deduplicateByURL(results)
	if len(deduped) != 2 {
		t.Fatalf("expected 2 results, got %d", len(deduped))
	}
	if deduped[0].Title != "First" {
		t.Errorf("expected first result 'First', got %q", deduped[0].Title)
	}
	if deduped[1].Title != "Second" {
		t.Errorf("expected second result 'Second', got %q", deduped[1].Title)
	}
}

func TestDeduplicateByURLEmpty(t *testing.T) {
	deduped := deduplicateByURL(nil)
	if len(deduped) != 0 {
		t.Fatalf("expected 0 results, got %d", len(deduped))
	}
}
