package app

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"power/internal/api"
	"power/internal/db"
)

func NewHTTPServer(addr string) (*http.Server, error) {
	dataDir := filepath.Join(".", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	database, err := db.Open(dataDir)
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}
	engine := api.NewRouter(database)
	return &http.Server{Addr: addr, Handler: engine}, nil
}
