package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

const tavilyAPIURL = "https://api.tavily.com/search"

type searchProxyRequest struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
	TimeRange  string `json:"time_range,omitempty"`
}

// handleSearch proxies POST /v1/search to the Tavily API.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req searchProxyRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Build Tavily request with server-side API key
	tavilyBody, err := json.Marshal(map[string]interface{}{
		"api_key":             s.tavilyKey,
		"query":               req.Query,
		"max_results":         req.MaxResults,
		"search_depth":        "basic",
		"include_raw_content": true,
		"days":                timeRangeToDays(req.TimeRange),
	})
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	tavilyReq, err := http.NewRequestWithContext(r.Context(), "POST", tavilyAPIURL, bytes.NewReader(tavilyBody))
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	tavilyReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(tavilyReq)
	if err != nil {
		http.Error(w, `{"error":"upstream search failed"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Transform Tavily response to managed search response format
	var tavilyResp struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody) //nolint:errcheck
		return
	}

	if err := json.Unmarshal(respBody, &tavilyResp); err != nil {
		http.Error(w, `{"error":"failed to parse upstream response"}`, http.StatusBadGateway)
		return
	}

	type outResult struct {
		Title   string  `json:"title"`
		URL     string  `json:"url"`
		Content string  `json:"content"`
		Snippet string  `json:"snippet"`
		Source  string  `json:"source"`
		Score   float64 `json:"score"`
	}
	var out []outResult
	for _, r := range tavilyResp.Results {
		snippet := r.Content
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		host := extractHost(r.URL)
		out = append(out, outResult{
			Title:   r.Title,
			URL:     r.URL,
			Content: r.Content,
			Snippet: snippet,
			Source:  host,
			Score:   r.Score,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"results": out}) //nolint:errcheck
}

func timeRangeToDays(tr string) int {
	switch tr {
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

func extractHost(rawURL string) string {
	for _, prefix := range []string{"https://", "http://", "www."} {
		rawURL = trimPrefix(rawURL, prefix)
	}
	if i := indexOf(rawURL, "/"); i > 0 {
		return rawURL[:i]
	}
	return rawURL
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
