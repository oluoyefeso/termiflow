package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	ks := newTestKeyStore(t)
	s := &Server{
		keys:         ks,
		httpClient:   &http.Client{},
		anthropicKey: "test-anthropic-key",
		tavilyKey:    "test-tavily-key",
		adminSecret:  "test-admin-secret",
		mux:          http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func TestAuthMiddlewareValid(t *testing.T) {
	s := newTestServer(t)
	key, _ := s.keys.IssueKey("test")

	req := httptest.NewRequest("POST", "/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	// Just test that it passes auth (will fail at proxy level without real body)
	// 400/502 means auth passed; 401 means auth failed
	s.ServeHTTP(rr, req)
	if rr.Code == http.StatusUnauthorized {
		t.Errorf("valid key should not get 401, got %d", rr.Code)
	}
}

func TestAuthMiddlewareMissingHeader(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("POST", "/v1/messages", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("missing header should get 401, got %d", rr.Code)
	}
}

func TestAuthMiddlewareRevokedKey(t *testing.T) {
	s := newTestServer(t)
	key, _ := s.keys.IssueKey("to-revoke")
	s.keys.RevokeKey(key)

	req := httptest.NewRequest("POST", "/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("revoked key should get 401, got %d", rr.Code)
	}
}

func TestAdminMiddlewareForbidden(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("POST", "/admin/keys", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("missing admin secret should get 403, got %d", rr.Code)
	}
}

func TestHealthCheck(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("health check should return 200, got %d", rr.Code)
	}
}
