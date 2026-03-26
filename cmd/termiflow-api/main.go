package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/oluoyefeso/termiflow/internal/api"
)

func main() {
	// Load .env if present (local dev). Silently ignored in production.
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv, err := api.NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}
	defer srv.Close()

	fmt.Printf("termiflow-api listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
