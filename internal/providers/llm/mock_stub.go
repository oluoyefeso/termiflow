//go:build !mock

package llm

// MockProvider is not available in non-mock builds.
// Build with -tags mock to enable: go build -tags mock ./...
func NewMockProvider() Provider {
	return nil
}
