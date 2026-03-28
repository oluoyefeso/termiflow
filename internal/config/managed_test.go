package config

import (
	"os"
	"testing"
)

func TestIsManagedMode(t *testing.T) {
	// Clean state
	os.Unsetenv("TERMIFLOW_API_KEY")
	resetViper()

	t.Run("false when no key set", func(t *testing.T) {
		if IsManagedMode() {
			t.Error("expected managed mode to be false")
		}
	})

	t.Run("true when TERMIFLOW_API_KEY set", func(t *testing.T) {
		os.Setenv("TERMIFLOW_API_KEY", "tf_testkey")
		defer os.Unsetenv("TERMIFLOW_API_KEY")
		Load("") //nolint:errcheck // reload config with env
		if !IsManagedMode() {
			t.Error("expected managed mode to be true with TERMIFLOW_API_KEY set")
		}
	})
}
