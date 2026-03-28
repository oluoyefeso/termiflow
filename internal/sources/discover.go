// Package sources provides RSS/Atom feed autodiscovery for custom source subscriptions.
package sources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// FeedInfo contains the resolved feed metadata from autodiscovery.
type FeedInfo struct {
	FeedURL     string // Resolved RSS/Atom feed URL
	Title       string // Feed title from metadata
	Description string // Feed description
	ScrapeOnly  bool   // True if no feed was found but scraping is possible
}

var (
	// feedLinkRe matches <link> tags with RSS/Atom type attributes.
	feedLinkRe = regexp.MustCompile(`(?i)<link[^>]+type=["']application/(rss|atom)\+xml["'][^>]*>`)
	hrefRe     = regexp.MustCompile(`(?i)href=["']([^"']+)["']`)

	// rssContentTypes that indicate a direct feed URL.
	rssContentTypes = []string{
		"application/rss+xml",
		"application/atom+xml",
		"application/xml",
		"text/xml",
	}

	// commonFeedPaths to probe when autodiscovery fails.
	commonFeedPaths = []string{"/feed", "/atom.xml", "/rss"}
)

// discoverClient has short timeouts appropriate for autodiscovery probing.
var discoverClient = &http.Client{
	Timeout: 5 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

// probeClient has a shorter timeout for fallback path probes.
var probeClient = &http.Client{
	Timeout: 3 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

// Discover resolves a URL to an RSS/Atom feed.
// It accepts any URL (blog homepage, direct feed URL, etc.) and returns
// the resolved feed metadata. Total timeout budget: 15 seconds.
func Discover(ctx context.Context, rawURL string) (*FeedInfo, error) {
	// Use caller's context deadline if set, otherwise default 15s.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
	}

	// Normalize URL
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// Step 1: Fetch the URL
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	req.Header.Set("User-Agent", "termiflow/1.0 (feed discovery)")

	resp, err := discoverClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("URL returned %s", resp.Status)
	}

	// Read body (limited to 512KB for autodiscovery)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")

	// Step 2: Check if the URL itself is a feed
	if isFeedContentType(contentType) {
		title := extractFeedTitle(string(body))
		return &FeedInfo{
			FeedURL: rawURL,
			Title:   title,
		}, nil
	}

	// Step 3: Parse HTML for <link rel="alternate"> feed discovery
	bodyStr := string(body)
	if feedURL := findFeedLink(bodyStr, rawURL); feedURL != "" {
		info, err := fetchFeedMetadata(ctx, feedURL)
		if err == nil {
			return info, nil
		}
		// Link found but feed unreachable, try probing
	}

	// Step 4: Probe common feed paths
	baseURL := strings.TrimRight(rawURL, "/")
	for _, path := range commonFeedPaths {
		probeURL := baseURL + path
		info, err := fetchFeedMetadata(ctx, probeURL)
		if err == nil {
			return info, nil
		}
	}

	// Step 5: No feed found, suggest scraping
	return &FeedInfo{ScrapeOnly: true}, fmt.Errorf("no RSS/Atom feed found at %s", rawURL)
}

// isFeedContentType returns true if the content type indicates RSS/Atom.
func isFeedContentType(ct string) bool {
	ct = strings.ToLower(ct)
	for _, rssType := range rssContentTypes {
		if strings.Contains(ct, rssType) {
			return true
		}
	}
	return false
}

// findFeedLink extracts a feed URL from HTML <link> autodiscovery tags.
func findFeedLink(html, baseURL string) string {
	matches := feedLinkRe.FindAllString(html, 5)
	for _, match := range matches {
		hrefMatch := hrefRe.FindStringSubmatch(match)
		if len(hrefMatch) >= 2 {
			href := hrefMatch[1]
			return resolveURL(href, baseURL)
		}
	}
	return ""
}

// resolveURL resolves a potentially relative URL against a base URL.
func resolveURL(href, baseURL string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	base := strings.TrimRight(baseURL, "/")
	if strings.HasPrefix(href, "/") {
		// Extract scheme + host from base
		parts := strings.SplitN(base, "//", 2)
		if len(parts) == 2 {
			hostParts := strings.SplitN(parts[1], "/", 2)
			return parts[0] + "//" + hostParts[0] + href
		}
	}
	return base + "/" + strings.TrimLeft(href, "/")
}

// fetchFeedMetadata fetches a URL and verifies it's a valid feed.
func fetchFeedMetadata(ctx context.Context, feedURL string) (*FeedInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "termiflow/1.0 (feed discovery)")

	resp, err := probeClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, err
	}

	bodyStr := string(body)

	// Check if it looks like a feed (has RSS or Atom markers)
	ct := resp.Header.Get("Content-Type")
	if isFeedContentType(ct) || looksLikeFeed(bodyStr) {
		title := extractFeedTitle(bodyStr)
		return &FeedInfo{
			FeedURL: feedURL,
			Title:   title,
		}, nil
	}

	return nil, fmt.Errorf("not a valid feed")
}

// looksLikeFeed does a quick content check for RSS/Atom markers.
// Requires XML-like patterns, not just substring matches (avoids false positives from HTML class names).
func looksLikeFeed(content string) bool {
	lower := strings.ToLower(content[:min(len(content), 1000)])
	return strings.Contains(lower, "<rss ") || strings.Contains(lower, "<rss>") ||
		strings.Contains(lower, "<feed ") || strings.Contains(lower, "<feed>") ||
		strings.Contains(lower, "<channel>") || strings.Contains(lower, "<channel ")
}

// extractFeedTitle extracts the <title> from an RSS/Atom feed.
func extractFeedTitle(content string) string {
	// Look for <title> tag (works for both RSS and Atom)
	idx := strings.Index(content, "<title>")
	if idx == -1 {
		idx = strings.Index(content, "<title ")
	}
	if idx == -1 {
		return ""
	}

	start := strings.Index(content[idx:], ">")
	if start == -1 {
		return ""
	}
	start += idx + 1

	end := strings.Index(content[start:], "</title>")
	if end == -1 {
		return ""
	}

	title := strings.TrimSpace(content[start : start+end])
	// Strip CDATA wrapper if present
	title = strings.TrimPrefix(title, "<![CDATA[")
	title = strings.TrimSuffix(title, "]]>")
	title = strings.TrimSpace(title)

	// Truncate to 50 chars
	if utf8.RuneCountInString(title) > 50 {
		runes := []rune(title)
		title = string(runes[:47]) + "..."
	}

	return title
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
