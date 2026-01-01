package search

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Scraper struct {
	client    *http.Client
	userAgent string
}

func NewScraper(userAgent string, timeout int) *Scraper {
	if userAgent == "" {
		userAgent = "termiflow/1.0"
	}
	if timeout == 0 {
		timeout = 30
	}

	return &Scraper{
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		userAgent: userAgent,
	}
}

func (s *Scraper) Name() string {
	return "scraper"
}

func (s *Scraper) Available() bool {
	return true
}

func (s *Scraper) Scrape(ctx context.Context, url string) (*SearchResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// Extract title
	title := doc.Find("title").First().Text()
	title = strings.TrimSpace(title)

	// Try to extract meta description
	description := ""
	doc.Find("meta[name='description']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists {
			description = content
		}
	})

	// Try to extract article content
	content := ""

	// Try common article selectors
	articleSelectors := []string{
		"article",
		".article-content",
		".post-content",
		".entry-content",
		"main",
		".content",
	}

	for _, selector := range articleSelectors {
		articleEl := doc.Find(selector).First()
		if articleEl.Length() > 0 {
			content = articleEl.Text()
			break
		}
	}

	// Clean up content
	content = cleanText(content)
	if len(content) > 2000 {
		content = content[:2000]
	}

	return &SearchResult{
		Title:   title,
		URL:     url,
		Snippet: description,
		Content: content,
		Source:  "scraper",
	}, nil
}

func cleanText(text string) string {
	// Remove excessive whitespace
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, " ")
}
