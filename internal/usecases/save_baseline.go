package usecases

import (
	"context"
	"log/slog"

	"github.com/che1nov/gopulse/internal/domain"
)

type BaselineStorage interface {
	Save(ctx context.Context, path string, snapshot domain.Snapshot) error
	Load(ctx context.Context, path string) (domain.Snapshot, error)
	Exists(path string) bool
}

type SaveBaseline struct {
	runner  RunBenchmarks
	storage BaselineStorage
	logger  *slog.Logger
}

func NewSaveBaseline(runner RunBenchmarks, storage BaselineStorage, logger *slog.Logger) SaveBaseline {
	return SaveBaseline{runner: runner, storage: storage, logger: logger}
}

func (uc SaveBaseline) Execute(ctx context.Context, cfg Config) (domain.Snapshot, error) {
	snapshot, err := uc.runner.Execute(ctx, cfg)
	if err != nil {
		return domain.Snapshot{}, err
	}

	if err := uc.storage.Save(ctx, cfg.BaselinePath, snapshot); err != nil {
		uc.logger.ErrorContext(ctx, "save baseline failed", "err", err, "operation", "save_baseline")
		return domain.Snapshot{}, err
	}

	uc.logger.InfoContext(ctx, "baseline saved", "path", cfg.BaselinePath, "operation", "save_baseline")
	return snapshot, nil
}
