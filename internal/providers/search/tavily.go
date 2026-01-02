package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const tavilyAPIURL = "https://api.tavily.com/search"

type TavilyProvider struct {
	apiKey string
	client *http.Client
}

func NewTavilyProvider(apiKey string) *TavilyProvider {
	return &TavilyProvider{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (p *TavilyProvider) Name() string {
	return "tavily"
}

func (p *TavilyProvider) Available() bool {
	return p.apiKey != ""
}

type tavilyRequest struct {
	APIKey            string `json:"api_key"`
	Query             string `json:"query"`
	SearchDepth       string `json:"search_depth,omitempty"`
	IncludeAnswer     bool   `json:"include_answer,omitempty"`
	IncludeRawContent bool   `json:"include_raw_content,omitempty"`
	MaxResults        int    `json:"max_results,omitempty"`
	Days              int    `json:"days,omitempty"`
}

type tavilyResponse struct {
	Answer  string `json:"answer"`
	Results []struct {
		Title   string  `json:"title"`
		URL     string  `json:"url"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
}

func (p *TavilyProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	maxResults := req.MaxResults
	if maxResults == 0 {
		maxResults = 5
	}

	days := timeRangeToDays(req.TimeRange)

	body := tavilyRequest{
		APIKey:            p.apiKey,
		Query:             req.Query,
		SearchDepth:       "advanced",
		IncludeAnswer:     false,
		IncludeRawContent: false,
		MaxResults:        maxResults,
		Days:              days,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", tavilyAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Tavily API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var tavilyResp tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tavilyResp); err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(tavilyResp.Results))
	for i, r := range tavilyResp.Results {
		results[i] = SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
			Source:  "tavily",
		}
	}

	return results, nil
}

func timeRangeToDays(timeRange string) int {
	switch timeRange {
	case "day":
		return 1
	case "week":
		return 7
	case "month":
		return 30
	case "year":
		return 365
	default:
		return 7
	}
}
