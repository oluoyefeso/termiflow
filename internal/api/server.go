package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// Server is the termiflow managed API backend.
//
//	Routes:
//	  POST   /v1/messages        — LLM proxy (auth required)
//	  POST   /v1/search          — Search proxy (auth required)
//	  GET    /health             — Health check
//	  POST   /admin/keys         — Issue key (admin secret required)
//	  DELETE /admin/keys/{key}   — Revoke key (admin secret required)
//	  GET    /admin/keys         — List keys (admin secret required)
type Server struct {
	keys         *KeyStore
	httpClient   *http.Client
	anthropicKey string
	tavilyKey    string
	adminSecret  string
	mux          *http.ServeMux
}

func NewServer() (*Server, error) {
	anthropicKey := os.Getenv("TERMIFLOW_ANTHROPIC_API_KEY")
	if anthropicKey == "" {
		return nil, fmt.Errorf("TERMIFLOW_ANTHROPIC_API_KEY environment variable is required")
	}

	tavilyKey := os.Getenv("TERMIFLOW_TAVILY_API_KEY")
	if tavilyKey == "" {
		return nil, fmt.Errorf("TERMIFLOW_TAVILY_API_KEY environment variable is required")
	}

	adminSecret := os.Getenv("TERMIFLOW_ADMIN_SECRET")
	if adminSecret == "" {
		return nil, fmt.Errorf("TERMIFLOW_ADMIN_SECRET environment variable is required")
	}

	dbPath := os.Getenv("TERMIFLOW_DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/keys.db"
	}

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	keys, err := NewKeyStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize key store: %w", err)
	}

	s := &Server{
		keys:         keys,
		httpClient:   &http.Client{},
		anthropicKey: anthropicKey,
		tavilyKey:    tavilyKey,
		adminSecret:  adminSecret,
		mux:          http.NewServeMux(),
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) Close() error {
	return s.keys.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	// Health check — no auth
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	// API routes — require valid termiflow key
	authMux := http.NewServeMux()
	authMux.HandleFunc("/v1/messages", s.handleMessages)
	authMux.HandleFunc("/v1/search", s.handleSearch)
	s.mux.Handle("/v1/", s.authMiddleware(authMux))

	// Admin routes — require admin secret
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/admin/keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			s.handleIssueKey(w, r)
		case http.MethodGet:
			s.handleListKeys(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/admin/keys/", s.handleRevokeKey)
	s.mux.Handle("/admin/", s.adminMiddleware(adminMux))
}
