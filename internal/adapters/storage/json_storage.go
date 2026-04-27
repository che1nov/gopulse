package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/che1nov/gopulse/internal/domain"
)

type JSONStorage struct{}

func NewJSONStorage() JSONStorage {
	return JSONStorage{}
}

func (s JSONStorage) Save(_ context.Context, path string, snapshot domain.Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func (s JSONStorage) Load(_ context.Context, path string) (domain.Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Snapshot{}, err
	}

	var snapshot domain.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return domain.Snapshot{}, err
	}
	return snapshot, nil
}

func (s JSONStorage) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
