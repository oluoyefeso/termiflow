package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDiscoverDirectFeedURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprint(w, `<?xml version="1.0"?><rss><channel><title>Test Blog</title></channel></rss>`)
	}))
	defer srv.Close()

	info, err := Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.FeedURL != srv.URL {
		t.Errorf("expected FeedURL=%s, got %s", srv.URL, info.FeedURL)
	}
	if info.Title != "Test Blog" {
		t.Errorf("expected Title='Test Blog', got %q", info.Title)
	}
}

func TestDiscoverHTMLWithLinkTag(t *testing.T) {
	var feedURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><head><link rel="alternate" type="application/rss+xml" href="%s/feed.xml"></head></html>`, feedURL)
	})
	mux.HandleFunc("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprint(w, `<?xml version="1.0"?><rss><channel><title>Discovered Blog</title></channel></rss>`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	feedURL = srv.URL

	info, err := Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Title != "Discovered Blog" {
		t.Errorf("expected Title='Discovered Blog', got %q", info.Title)
	}
	if info.FeedURL != srv.URL+"/feed.xml" {
		t.Errorf("expected FeedURL ending in /feed.xml, got %s", info.FeedURL)
	}
}

func TestDiscoverProbeCommonPath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><title>Blog</title></head><body>No feed link</body></html>`)
	})
	mux.HandleFunc("/feed", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		fmt.Fprint(w, `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><title>Probed Feed</title></feed>`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	info, err := Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Title != "Probed Feed" {
		t.Errorf("expected Title='Probed Feed', got %q", info.Title)
	}
}

func TestDiscoverNoFeedScrapeHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>Just a website</body></html>`)
	}))
	defer srv.Close()

	info, err := Discover(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for site with no feed")
	}
	if info == nil || !info.ScrapeOnly {
		t.Error("expected ScrapeOnly=true hint")
	}
}

func TestDiscoverTimeout(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(30 * time.Second):
		}
	}))
	srv.Start()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := Discover(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestDiscoverInvalidURL(t *testing.T) {
	_, err := Discover(context.Background(), "http://localhost:1")
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
}

func TestDiscoverAtomLinkTag(t *testing.T) {
	var feedURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><head><link rel="alternate" type="application/atom+xml" href="%s/atom.xml" title="Atom Feed"></head></html>`, feedURL)
	})
	mux.HandleFunc("/atom.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		fmt.Fprint(w, `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><title>Atom Blog</title></feed>`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	feedURL = srv.URL

	info, err := Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Title != "Atom Blog" {
		t.Errorf("expected Title='Atom Blog', got %q", info.Title)
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		href, base, want string
	}{
		{"https://example.com/feed", "https://example.com", "https://example.com/feed"},
		{"/feed.xml", "https://example.com/blog", "https://example.com/feed.xml"},
		{"feed.xml", "https://example.com/blog", "https://example.com/blog/feed.xml"},
		{"//cdn.example.com/feed", "https://example.com", "https://cdn.example.com/feed"},
	}
	for _, tt := range tests {
		got := resolveURL(tt.href, tt.base)
		if got != tt.want {
			t.Errorf("resolveURL(%q, %q) = %q, want %q", tt.href, tt.base, got, tt.want)
		}
	}
}

func TestExtractFeedTitleCDATA(t *testing.T) {
	content := `<?xml version="1.0"?><rss><channel><title><![CDATA[My Cool Blog]]></title></channel></rss>`
	title := extractFeedTitle(content)
	if title != "My Cool Blog" {
		t.Errorf("expected 'My Cool Blog', got %q", title)
	}
}

func TestExtractFeedTitleLong(t *testing.T) {
	long := "This Is A Very Long Blog Title That Exceeds Fifty Characters In Total Length"
	content := fmt.Sprintf(`<rss><channel><title>%s</title></channel></rss>`, long)
	title := extractFeedTitle(content)
	if len([]rune(title)) > 50 {
		t.Errorf("expected title truncated to 50 chars, got %d", len([]rune(title)))
	}
}
