package model

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cockroachdb/pebble"
)

func OpenDB(dbDir string) (*pebble.DB, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbDir), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := pebble.Open(dbDir, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble db: %w", err)
	}

	return db, nil
}
