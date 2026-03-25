package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ManagedSearchProvider calls the termiflow backend proxy instead of Tavily directly.
//
//	CLI ──► POST {baseURL}/v1/search ──► termiflow backend ──► Tavily
//	        Authorization: Bearer {apiKey}      api_key: server-side key
type ManagedSearchProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewManagedSearchProvider(apiKey, baseURL string) *ManagedSearchProvider {
	if baseURL == "" {
		baseURL = "https://api.termiflow.com"
	}
	return &ManagedSearchProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (p *ManagedSearchProvider) Name() string    { return "managed" }
func (p *ManagedSearchProvider) Available() bool { return p.apiKey != "" }

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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/search", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("managed search: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("managed search: API error %s: %s", resp.Status, string(bodyBytes))
	}

	var raw managedSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("managed search: failed to decode response: %w", err)
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
