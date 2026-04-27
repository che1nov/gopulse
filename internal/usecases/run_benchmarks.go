package usecases

import (
	"context"
	"log/slog"
	"time"

	"github.com/che1nov/gopulse/internal/domain"
)

type BenchmarkRunner interface {
	Run(ctx context.Context, cfg BenchmarkConfig) ([]domain.Benchmark, error)
	Project(ctx context.Context) string
	GoVersion(ctx context.Context) string
}

type RunBenchmarks struct {
	runner BenchmarkRunner
	logger *slog.Logger
}

func NewRunBenchmarks(runner BenchmarkRunner, logger *slog.Logger) RunBenchmarks {
	return RunBenchmarks{runner: runner, logger: logger}
}

func (uc RunBenchmarks) Execute(ctx context.Context, cfg Config) (domain.Snapshot, error) {
	uc.logger.InfoContext(ctx, "run benchmarks", "packages", cfg.Benchmark.Packages, "operation", "run_benchmarks")

	benchmarks, err := uc.runner.Run(ctx, cfg.Benchmark)
	if err != nil {
		uc.logger.ErrorContext(ctx, "benchmarks failed", "err", err, "operation", "run_benchmarks")
		return domain.Snapshot{}, err
	}

	return domain.Snapshot{
		Project:    uc.runner.Project(ctx),
		GoVersion:  uc.runner.GoVersion(ctx),
		CreatedAt:  time.Now().UTC(),
		Benchmarks: benchmarks,
	}, nil
}
