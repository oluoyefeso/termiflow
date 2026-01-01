package models

import (
	"testing"
)

func TestDefaultCategories(t *testing.T) {
	// Verify we have the expected number of categories
	expectedCount := 6
	if len(DefaultCategories) != expectedCount {
		t.Errorf("DefaultCategories count = %d, want %d", len(DefaultCategories), expectedCount)
	}

	// Verify expected categories exist
	expectedNames := []string{
		"silicon-chips",
		"rust-lang",
		"llm-inference",
		"webgpu",
		"systems-programming",
		"kubernetes",
	}

	for _, name := range expectedNames {
		found := false
		for _, cat := range DefaultCategories {
			if cat.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected category %q not found", name)
		}
	}
}

func TestDefaultCategories_HaveRequiredFields(t *testing.T) {
	for _, cat := range DefaultCategories {
		if cat.Name == "" {
			t.Error("Category has empty Name")
		}
		if cat.DisplayName == "" {
			t.Errorf("Category %q has empty DisplayName", cat.Name)
		}
		if cat.Description == "" {
			t.Errorf("Category %q has empty Description", cat.Name)
		}
		if len(cat.Keywords) == 0 {
			t.Errorf("Category %q has no Keywords", cat.Name)
		}
	}
}

func TestGetCategoryByName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectFound bool
		expectName  string
	}{
		{
			name:        "find silicon-chips",
			input:       "silicon-chips",
			expectFound: true,
			expectName:  "silicon-chips",
		},
		{
			name:        "find rust-lang",
			input:       "rust-lang",
			expectFound: true,
			expectName:  "rust-lang",
		},
		{
			name:        "find kubernetes",
			input:       "kubernetes",
			expectFound: true,
			expectName:  "kubernetes",
		},
		{
			name:        "not found",
			input:       "nonexistent",
			expectFound: false,
		},
		{
			name:        "empty string",
			input:       "",
			expectFound: false,
		},
		{
			name:        "case sensitive",
			input:       "RUST-LANG",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCategoryByName(tt.input)

			if tt.expectFound {
				if result == nil {
					t.Errorf("GetCategoryByName(%q) = nil, expected category", tt.input)
					return
				}
				if result.Name != tt.expectName {
					t.Errorf("GetCategoryByName(%q).Name = %q, want %q", tt.input, result.Name, tt.expectName)
				}
			} else {
				if result != nil {
					t.Errorf("GetCategoryByName(%q) = %v, expected nil", tt.input, result)
				}
			}
		})
	}
}

func TestCategory_SiliconChips(t *testing.T) {
	cat := GetCategoryByName("silicon-chips")
	if cat == nil {
		t.Fatal("silicon-chips category not found")
	}

	if cat.DisplayName != "Silicon & Semiconductors" {
		t.Errorf("DisplayName = %q, want %q", cat.DisplayName, "Silicon & Semiconductors")
	}

	// Should have RSS feeds
	if len(cat.DefaultRSS) == 0 {
		t.Error("silicon-chips should have DefaultRSS feeds")
	}

	// Check specific keywords
	hasKeyword := func(keyword string) bool {
		for _, k := range cat.Keywords {
			if k == keyword {
				return true
			}
		}
		return false
	}

	expectedKeywords := []string{"semiconductor", "TSMC", "Intel"}
	for _, kw := range expectedKeywords {
		if !hasKeyword(kw) {
			t.Errorf("silicon-chips missing expected keyword %q", kw)
		}
	}
}

func TestCategory_RustLang(t *testing.T) {
	cat := GetCategoryByName("rust-lang")
	if cat == nil {
		t.Fatal("rust-lang category not found")
	}

	if cat.DisplayName != "Rust Programming" {
		t.Errorf("DisplayName = %q, want %q", cat.DisplayName, "Rust Programming")
	}

	// Check specific keywords
	hasKeyword := func(keyword string) bool {
		for _, k := range cat.Keywords {
			if k == keyword {
				return true
			}
		}
		return false
	}

	expectedKeywords := []string{"rust programming", "crates.io"}
	for _, kw := range expectedKeywords {
		if !hasKeyword(kw) {
			t.Errorf("rust-lang missing expected keyword %q", kw)
		}
	}
}

func TestCategory_LLMInference(t *testing.T) {
	cat := GetCategoryByName("llm-inference")
	if cat == nil {
		t.Fatal("llm-inference category not found")
	}

	if cat.DisplayName != "LLM & AI Inference" {
		t.Errorf("DisplayName = %q, want %q", cat.DisplayName, "LLM & AI Inference")
	}

	// Check specific keywords
	hasKeyword := func(keyword string) bool {
		for _, k := range cat.Keywords {
			if k == keyword {
				return true
			}
		}
		return false
	}

	expectedKeywords := []string{"LLM inference", "vLLM", "quantization"}
	for _, kw := range expectedKeywords {
		if !hasKeyword(kw) {
			t.Errorf("llm-inference missing expected keyword %q", kw)
		}
	}
}

func TestCategory_ReturnsPointer(t *testing.T) {
	// Ensure modifications don't affect the original
	cat1 := GetCategoryByName("rust-lang")
	cat1.DisplayName = "Modified"

	cat2 := GetCategoryByName("rust-lang")
	if cat2.DisplayName == "Modified" {
		t.Error("GetCategoryByName should return independent copies")
	}
}
