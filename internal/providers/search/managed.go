package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/oluoyefeso/termiflow/internal/providers"
)

// ManagedSearchProvider calls the termiflow backend proxy instead of Tavily directly.
//
//	CLI ──► POST {baseURL}/v1/search ──► termiflow backend ──► Tavily
//	        Authorization: Bearer {apiKey}      api_key: server-side key
type ManagedSearchProvider struct {
	mc *providers.ManagedClient
}

func NewManagedSearchProvider(apiKey, baseURL string) *ManagedSearchProvider {
	return &ManagedSearchProvider{
		mc: providers.NewManagedClient(apiKey, baseURL),
	}
}

func (p *ManagedSearchProvider) Name() string    { return "managed" }
func (p *ManagedSearchProvider) Available() bool { return p.mc.APIKey != "" }

type managedSearchRequest struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
	TimeRange  string `json:"time_range,omitempty"`
}

type managedSearchResponse struct {
	Results []struct {
		Title         string  `json:"title"`
		URL           string  `json:"url"`
		Content       string  `json:"content"`
		Snippet       string  `json:"snippet"`
		PublishedDate string  `json:"published_date"`
		Source        string  `json:"source"`
		Score         float64 `json:"score"`
	} `json:"results"`
}

func (p *ManagedSearchProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	body := managedSearchRequest(req)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := providers.DoWithRetry(ctx, func() (*http.Response, error) { //nolint:bodyclose // DoWithRetry manages body lifecycle
		httpReq, err := p.mc.NewRequest(ctx, "POST", "/v1/search", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}
		return p.mc.Do(httpReq)
	})
	if err != nil {
		return nil, fmt.Errorf("managed search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("managed search: API error %s: %s", resp.Status, string(bodyBytes))
	}

	var raw managedSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("managed search: unexpected response from Termiflow API (status: %d) — try again or check api.termiflow.com/health", resp.StatusCode)
	}

	var results []SearchResult
	for _, r := range raw.Results {
		pub, _ := time.Parse(time.RFC3339, r.PublishedDate)
		results = append(results, SearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Content:     r.Content,
			Snippet:     r.Snippet,
			PublishedAt: pub,
			Source:      r.Source,
		})
	}
	return results, nil
}
