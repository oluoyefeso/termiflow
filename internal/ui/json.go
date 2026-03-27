package ui

import (
	"encoding/json"
	"os"
	"time"
)

// JSONEnvelope is the common wrapper for all --json output.
type JSONEnvelope struct {
	Data  any      `json:"data"`
	Meta  JSONMeta `json:"meta"`
	Error string   `json:"error,omitempty"`
}

// JSONMeta contains metadata included in every JSON response.
type JSONMeta struct {
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

// WriteJSON marshals data in the standard envelope to stdout.
func WriteJSON(data any, version string) error {
	env := JSONEnvelope{
		Data: data,
		Meta: JSONMeta{
			Version:   version,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}

// WriteJSONError marshals an error response with optional partial data.
func WriteJSONError(data any, errMsg string, version string) error {
	env := JSONEnvelope{
		Data:  data,
		Error: errMsg,
		Meta: JSONMeta{
			Version:   version,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}
