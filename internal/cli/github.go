package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/oluoyefeso/termiflow/internal/config"
)

const maxResponseSize = 5 * 1024 * 1024 // 5MB

// githubRelease represents a release from the GitHub Releases API.
type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
}

const githubReleasesURL = "https://api.github.com/repos/oluoyefeso/termiflow/releases"

// releasesCache stores cached GitHub releases with ETag for conditional requests.
type releasesCache struct {
	ETag     string          `json:"etag"`
	Data     []githubRelease `json:"data"`
	CachedAt time.Time       `json:"cached_at"`
}

// githubClient returns an HTTP client with a 10-second timeout.
func githubClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// fetchLatestRelease fetches the latest release from GitHub, using ETag caching.
func fetchLatestRelease() (*githubRelease, error) {
	releases, err := fetchReleasesFromAPI(githubReleasesURL+"/latest", "releases-latest", true)
	if err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, nil
	}
	return &releases[0], nil
}

// fetchReleases fetches up to limit releases from GitHub, using ETag caching.
func fetchReleases(limit int) ([]githubRelease, error) {
	if limit < 1 {
		limit = 1
	} else if limit > 100 {
		limit = 100
	}
	url := fmt.Sprintf("%s?per_page=%d", githubReleasesURL, limit)
	cacheKey := fmt.Sprintf("releases-%d", limit)
	return fetchReleasesFromAPI(url, cacheKey, false)
}

// fetchReleasesFromAPI handles the actual HTTP request with ETag caching.
// If single is true, the response is a single object (not an array).
func fetchReleasesFromAPI(url, cacheKey string, single bool) ([]githubRelease, error) {
	cache := loadReleasesCache(cacheKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if cache != nil && cache.ETag != "" {
		req.Header.Set("If-None-Match", cache.ETag)
	}

	resp, err := githubClient().Do(req)
	if err != nil {
		// Network error — return cache if available
		if cache != nil && len(cache.Data) > 0 {
			return cache.Data, nil
		}
		return nil, fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer resp.Body.Close()

	// 304 Not Modified — return cached data
	if resp.StatusCode == http.StatusNotModified && cache != nil {
		return cache.Data, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode == http.StatusForbidden {
		// Rate limited — return cache if available
		if cache != nil && len(cache.Data) > 0 {
			return cache.Data, nil
		}
		return nil, fmt.Errorf("GitHub API rate limited — try again in a few minutes")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	limitedBody := io.LimitReader(resp.Body, maxResponseSize)

	var releases []githubRelease
	if single {
		var release githubRelease
		if err := json.NewDecoder(limitedBody).Decode(&release); err != nil {
			return nil, fmt.Errorf("parse release: %w", err)
		}
		releases = []githubRelease{release}
	} else {
		if err := json.NewDecoder(limitedBody).Decode(&releases); err != nil {
			return nil, fmt.Errorf("parse releases: %w", err)
		}
	}

	// Save to cache with ETag
	etag := resp.Header.Get("ETag")
	saveReleasesCache(cacheKey, &releasesCache{
		ETag:     etag,
		Data:     releases,
		CachedAt: time.Now(),
	})

	return releases, nil
}

// loadReleasesCache loads a cached releases response from disk.
func loadReleasesCache(key string) *releasesCache {
	path := releasesCachePath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cache releasesCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil // Corrupt cache — ignore
	}
	return &cache
}

// saveReleasesCache writes a releases cache to disk.
func saveReleasesCache(key string, cache *releasesCache) {
	path := releasesCachePath(key)
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0600)
}

// releasesCachePath returns the file path for a releases cache entry.
func releasesCachePath(key string) string {
	return filepath.Join(config.GetCacheDir(), key+".json")
}
