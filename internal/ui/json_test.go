package ui

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"key": "value"}
	err := WriteJSON(data, "1.0.0")
	w.Close() //nolint:errcheck
	os.Stdout = old

	if err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var env JSONEnvelope
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, output)
	}

	if env.Meta.Version != "1.0.0" {
		t.Errorf("meta.version = %q, want %q", env.Meta.Version, "1.0.0")
	}
	if env.Meta.Timestamp == "" {
		t.Error("meta.timestamp should not be empty")
	}
	if env.Error != "" {
		t.Errorf("error should be empty, got %q", env.Error)
	}
}

func TestWriteJSONNilData(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := WriteJSON(nil, "dev")
	w.Close() //nolint:errcheck
	os.Stdout = old

	if err != nil {
		t.Fatalf("WriteJSON with nil returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var env JSONEnvelope
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if env.Data != nil {
		t.Errorf("data should be nil for nil input, got %v", env.Data)
	}
}

func TestWriteJSONError(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"partial": "content"}
	err := WriteJSONError(data, "stream failed", "1.0.0")
	w.Close() //nolint:errcheck
	os.Stdout = old

	if err != nil {
		t.Fatalf("WriteJSONError returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var env JSONEnvelope
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if env.Error != "stream failed" {
		t.Errorf("error = %q, want %q", env.Error, "stream failed")
	}
}

func TestWriteJSONNoANSICodes(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{
		"title":   "Test Article",
		"summary": "A plain text summary with no styling",
	}
	err := WriteJSON(data, "1.0.0")
	w.Close() //nolint:errcheck
	os.Stdout = old

	if err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if strings.Contains(output, "\x1b") {
		t.Errorf("JSON output contains ANSI escape codes: %q", output)
	}
	if strings.Contains(output, "\033") {
		t.Errorf("JSON output contains ANSI escape codes (octal): %q", output)
	}
}
