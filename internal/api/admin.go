package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleIssueKey handles POST /admin/keys — issues a new API key.
func (s *Server) handleIssueKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck

	key, err := s.keys.IssueKey(req.Name)
	if err != nil {
		http.Error(w, `{"error":"failed to issue key"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"key": key, "name": req.Name}) //nolint:errcheck
}

// handleRevokeKey handles DELETE /admin/keys/{key} — revokes a key.
func (s *Server) handleRevokeKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/admin/keys/")
	if key == "" {
		http.Error(w, `{"error":"key required"}`, http.StatusBadRequest)
		return
	}

	if err := s.keys.RevokeKey(key); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "revoked", "key": key}) //nolint:errcheck
}

// handleListKeys handles GET /admin/keys — lists all keys.
func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	keys, err := s.keys.ListKeys()
	if err != nil {
		http.Error(w, `{"error":"failed to list keys"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"keys": keys}) //nolint:errcheck
}
