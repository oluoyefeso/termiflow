package api

import (
	"net/http"
	"strings"
)

// authMiddleware validates the Bearer token in the Authorization header.
// Returns 401 if missing, invalid, or revoked.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"missing or invalid Authorization header"}`, http.StatusUnauthorized)
			return
		}

		key := strings.TrimPrefix(authHeader, "Bearer ")
		valid, err := s.keys.ValidateKey(key)
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		if !valid {
			http.Error(w, `{"error":"invalid or revoked API key"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// adminMiddleware validates the X-Admin-Secret header for admin endpoints.
func (s *Server) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Admin-Secret") != s.adminSecret {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
