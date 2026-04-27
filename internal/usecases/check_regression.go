package usecases

import (
	"context"
	"log/slog"

	"github.com/che1nov/gopulse/internal/domain"
)

type CheckRegression struct {
	runner  RunBenchmarks
	storage BaselineStorage
	logger  *slog.Logger
}

func NewCheckRegression(runner RunBenchmarks, storage BaselineStorage, logger *slog.Logger) CheckRegression {
	return CheckRegression{runner: runner, storage: storage, logger: logger}
}

func (uc CheckRegression) Execute(ctx context.Context, cfg Config) (domain.CheckResult, error) {
	baseline, err := uc.storage.Load(ctx, cfg.BaselinePath)
	if err != nil {
		uc.logger.ErrorContext(ctx, "load baseline failed", "path", cfg.BaselinePath, "err", err, "operation", "check_regression")
		return domain.CheckResult{}, err
	}

	current, err := uc.runner.Execute(ctx, cfg)
	if err != nil {
		return domain.CheckResult{}, err
	}

	result := domain.CompareSnapshots(baseline, current, cfg.Thresholds)
	if result.Failed {
		uc.logger.WarnContext(ctx, "regression detected", "operation", "check_regression")
	}
	return result, nil
}
