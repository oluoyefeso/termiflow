//go:build !mock

package search

func NewMockSearchProvider() Provider {
	return nil
}
