package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/semver/v3"
)

const cacheTTL = time.Hour

// Announcement matches the API response shape.
type Announcement struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Message   string  `json:"message"`
	ActionURL string  `json:"action_url,omitempty"`
	CreatedAt string  `json:"created_at"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// Cache is the local cache file structure.
type Cache struct {
	Announcements       []Announcement `json:"announcements"`
	LatestVersion       string         `json:"latest_version"`
	MinSupportedVersion string         `json:"min_supported_version"`
	FetchedAt           string         `json:"fetched_at"`
}

// DismissedState tracks which announcements and versions the user has seen.
type DismissedState struct {
	DismissedIDs      map[string]string `json:"dismissed_ids"`      // id → expires_at
	DismissedVersions []string          `json:"dismissed_versions"` // versions already shown
}

// BannerInfo represents a banner to display.
type BannerInfo struct {
	Type    string // info, warning, update, breaking, version
	Message string
}

// Manager handles notification caching, display, and background fetch.
type Manager struct {
	baseURL       string
	apiKey        string
	version       string
	cacheDir      string
	isManagedMode bool

	cache     Cache
	dismissed DismissedState
	stale     bool
}

// NewManager creates a notification manager.
func NewManager(baseURL, apiKey, version, cacheDir string, isManagedMode bool) *Manager {
	if baseURL == "" {
		baseURL = "https://api.termiflow.com"
	}
	return &Manager{
		baseURL:       baseURL,
		apiKey:        apiKey,
		version:       version,
		cacheDir:      cacheDir,
		isManagedMode: isManagedMode,
		dismissed: DismissedState{
			DismissedIDs: make(map[string]string),
		},
	}
}

// LoadCache reads the cache and dismissed state from disk.
func (m *Manager) LoadCache() {
	m.loadCacheFile()
	m.loadDismissedFile()
}

func (m *Manager) loadCacheFile() {
	m.stale = false // Reset before re-evaluating
	path := filepath.Join(m.cacheDir, "announcements.json")
	data, err := os.ReadFile(path)
	if err != nil {
		m.stale = true
		return
	}
	if err := json.Unmarshal(data, &m.cache); err != nil {
		// Corrupted cache — treat as empty
		m.cache = Cache{}
		m.stale = true
		return
	}
	// Check TTL
	if m.cache.FetchedAt != "" {
		if fetched, err := time.Parse(time.RFC3339, m.cache.FetchedAt); err == nil {
			if time.Since(fetched) > cacheTTL {
				m.stale = true
			}
		}
	}
}

func (m *Manager) loadDismissedFile() {
	path := filepath.Join(m.cacheDir, "dismissed.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, &m.dismissed); err != nil {
		m.dismissed = DismissedState{DismissedIDs: make(map[string]string)}
	}
	if m.dismissed.DismissedIDs == nil {
		m.dismissed.DismissedIDs = make(map[string]string)
	}
}

// GetBanners returns banners to display from the loaded cache.
func (m *Manager) GetBanners() []BannerInfo {
	var banners []BannerInfo

	currentVer, _ := semver.NewVersion(m.version)
	isDev := m.version == "dev" || m.version == "" || currentVer == nil

	// Check min_supported_version — always shown if below minimum
	if !isDev && m.cache.MinSupportedVersion != "" {
		if minVer, err := semver.NewVersion(m.cache.MinSupportedVersion); err == nil {
			if currentVer.LessThan(minVer) {
				banners = append(banners, BannerInfo{
					Type:    "breaking",
					Message: fmt.Sprintf("Your version (v%s) is no longer supported. Run termiflow upgrade.", m.version),
				})
			}
		}
	}

	// Check latest_version — shown once per version (skip for dev builds)
	if !isDev && m.cache.LatestVersion != "" {
		if latestVer, err := semver.NewVersion(m.cache.LatestVersion); err == nil {
			if currentVer.LessThan(latestVer) {
				if !m.isVersionDismissed(m.cache.LatestVersion) {
					banners = append(banners, BannerInfo{
						Type:    "version",
						Message: fmt.Sprintf("Update available: v%s → v%s · run termiflow upgrade", m.version, m.cache.LatestVersion),
					})
				}
			}
		}
	}

	// Show all active announcements (shown every run until they expire)
	for _, ann := range m.cache.Announcements {
		// Skip expired announcements still in cache
		if ann.ExpiresAt != nil {
			if t, err := time.Parse(time.RFC3339, *ann.ExpiresAt); err == nil && time.Now().UTC().After(t) {
				continue
			}
		}
		banners = append(banners, BannerInfo{
			Type:    ann.Type,
			Message: ann.Message,
		})
	}

	return banners
}

// MarkDisplayed marks version update banners as dismissed so they show once per version.
// Announcements are NOT dismissed — they show on every run until they expire on the server.
func (m *Manager) MarkDisplayed(banners []BannerInfo) {
	changed := false
	for _, b := range banners {
		if b.Type == "version" {
			m.dismissed.DismissedVersions = append(m.dismissed.DismissedVersions, m.cache.LatestVersion)
			changed = true
		}
	}
	if changed {
		m.saveDismissedFile()
	}
}

func (m *Manager) isVersionDismissed(version string) bool {
	for _, v := range m.dismissed.DismissedVersions {
		if v == version {
			return true
		}
	}
	return false
}

func (m *Manager) saveDismissedFile() {
	data, err := json.MarshalIndent(m.dismissed, "", "  ")
	if err != nil {
		return
	}
	path := filepath.Join(m.cacheDir, "dismissed.json")
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return
	}
	os.Rename(tmpPath, path) //nolint:errcheck
}

// IsStale returns true if the cache is missing, corrupted, or expired.
func (m *Manager) IsStale() bool {
	return m.stale
}

// Fetch fetches announcements synchronously and updates the cache.
func (m *Manager) Fetch(ctx context.Context) {
	if m.isManagedMode {
		m.fetchFromAPI(ctx)
	} else {
		m.fetchFromGitHub(ctx)
	}
	// Reload cache from disk after fetch
	m.loadCacheFile()
}

// GetLatestVersion returns the cached latest version, if available.
func (m *Manager) GetLatestVersion() string {
	return m.cache.LatestVersion
}

// FetchAsync fetches announcements in the background and updates the cache.
func (m *Manager) FetchAsync(ctx context.Context) {
	if m.isManagedMode {
		m.fetchFromAPI(ctx)
	} else {
		m.fetchFromGitHub(ctx)
	}
}

func (m *Manager) fetchFromAPI(ctx context.Context) {
	endpoint := fmt.Sprintf("%s/v1/announcements", m.baseURL)
	if _, err := semver.NewVersion(m.version); err == nil {
		endpoint += "?version=" + url.QueryEscape(m.version)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		return // Network error — silently use stale cache
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return // Older API version — no announcements endpoint
	}
	if resp.StatusCode != http.StatusOK {
		return // Rate limited or other error — silently skip
	}

	var apiResp struct {
		Announcements       []Announcement `json:"announcements"`
		LatestVersion       string         `json:"latest_version"`
		MinSupportedVersion string         `json:"min_supported_version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return
	}

	cache := Cache{
		Announcements:       apiResp.Announcements,
		LatestVersion:       apiResp.LatestVersion,
		MinSupportedVersion: apiResp.MinSupportedVersion,
		FetchedAt:           time.Now().UTC().Format(time.RFC3339),
	}
	m.writeCache(cache)
	m.pruneExpiredDismissed()
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func (m *Manager) fetchFromGitHub(ctx context.Context) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://api.github.com/repos/oluoyefeso/termiflow/releases/latest", nil)
	if err != nil {
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return
	}

	latestVersion := release.TagName
	if len(latestVersion) > 0 && latestVersion[0] == 'v' {
		latestVersion = latestVersion[1:]
	}

	cache := Cache{
		LatestVersion: latestVersion,
		FetchedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	m.writeCache(cache)
}

func (m *Manager) writeCache(cache Cache) {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	os.MkdirAll(m.cacheDir, 0755) //nolint:errcheck
	path := filepath.Join(m.cacheDir, "announcements.json")
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return
	}
	os.Rename(tmpPath, path) //nolint:errcheck
}

func (m *Manager) pruneExpiredDismissed() {
	now := time.Now().UTC()
	changed := false
	for id, expiresAt := range m.dismissed.DismissedIDs {
		if expiresAt == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, expiresAt); err == nil && now.After(t) {
			delete(m.dismissed.DismissedIDs, id)
			changed = true
		}
	}
	if changed {
		m.saveDismissedFile()
	}
}
