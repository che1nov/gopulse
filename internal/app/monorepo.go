package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/che1nov/gopulse/internal/adapters/gotest"
	"github.com/che1nov/gopulse/internal/adapters/reporter"
	"github.com/che1nov/gopulse/internal/adapters/storage"
	"github.com/che1nov/gopulse/internal/usecases"
	"github.com/che1nov/gopulse/pkg/logger"
)

func monorepo(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: gopulse monorepo doctor|run")
		return 2
	}

	modules := findGoModules(".")
	if len(modules) == 0 {
		fmt.Fprintln(stderr, "no Go modules found")
		return 1
	}

	switch args[0] {
	case "doctor":
		return monorepoDoctor(ctx, modules, stdout, stderr)
	case "run":
		return monorepoRun(ctx, modules, stdout, stderr)
	case "baseline":
		if len(args) < 2 || args[1] != "save" {
			fmt.Fprintln(stderr, "usage: gopulse monorepo baseline save")
			return 2
		}
		return monorepoBaselineSave(ctx, modules, stdout, stderr)
	case "check":
		return monorepoCheck(ctx, modules, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown monorepo command: %s\n", args[0])
		return 2
	}
}

func monorepoDoctor(ctx context.Context, modules []string, stdout, stderr io.Writer) int {
	failed := false
	fmt.Fprintf(stdout, "Go modules found: %d\n", len(modules))
	for _, module := range modules {
		cfg, err := loadModuleConfig(module)
		if err != nil {
			fmt.Fprintf(stderr, "%s: load config: %v\n", module, err)
			failed = true
			continue
		}

		runner := gotest.NewRunner(module, logger.New(io.Discard, slog.LevelWarn))
		packages, packageErr := runner.Packages(ctx, cfg.Benchmark.Packages)
		benchmarks, benchmarkErr := runner.BenchmarkFileCount(ctx, cfg.Benchmark.Packages)

		statusText := "OK"
		if packageErr != nil || benchmarkErr != nil || benchmarks == 0 {
			statusText = "not ready"
			failed = true
		}

		packageCount := len(packages)
		if packageErr != nil {
			packageCount = 0
		}
		fmt.Fprintf(stdout, "%s\n  packages: %d\n  benchmarks: %d\n  status: %s\n", module, packageCount, benchmarks, statusText)
		if packageErr != nil {
			fmt.Fprintln(stderr, formatCommandError(module+" inspect packages", packageErr))
		}
		if benchmarkErr != nil {
			fmt.Fprintf(stderr, "%s inspect benchmarks: %v\n", module, benchmarkErr)
		}
	}

	if failed {
		return 1
	}
	return 0
}

func monorepoRun(ctx context.Context, modules []string, stdout, stderr io.Writer) int {
	failed := false
	for _, module := range modules {
		cfg, err := loadModuleConfig(module)
		if err != nil {
			fmt.Fprintf(stderr, "%s: load config: %v\n", module, err)
			failed = true
			continue
		}

		log := logger.New(io.Discard, slog.LevelWarn)
		runner := gotest.NewRunner(module, log)
		uc := usecases.NewRunBenchmarks(runner, log)
		snapshot, err := uc.Execute(ctx, cfg)
		if err != nil {
			fmt.Fprintln(stderr, formatCommandError(module+" run benchmarks", err))
			failed = true
			continue
		}

		fmt.Fprintf(stdout, "\n## %s\n", module)
		if err := reporter.NewTerminal().PrintSnapshot(stdout, snapshot); err != nil {
			fmt.Fprintf(stderr, "%s print report: %v\n", module, err)
			failed = true
		}
	}

	if failed {
		return 1
	}
	return 0
}

func monorepoBaselineSave(ctx context.Context, modules []string, stdout, stderr io.Writer) int {
	failed := false
	for _, module := range modules {
		cfg, err := loadModuleConfig(module)
		if err != nil {
			fmt.Fprintf(stderr, "%s: load config: %v\n", module, err)
			failed = true
			continue
		}

		log := logger.New(io.Discard, slog.LevelWarn)
		runner := gotest.NewRunner(module, log)
		store := storage.NewJSONStorage()
		runBenchmarks := usecases.NewRunBenchmarks(runner, log)
		uc := usecases.NewSaveBaseline(runBenchmarks, store, log)
		snapshot, err := uc.Execute(ctx, cfg)
		if err != nil {
			fmt.Fprintln(stderr, formatCommandError(module+" save baseline", err))
			failed = true
			continue
		}

		fmt.Fprintf(stdout, "%s: baseline saved (%d benchmarks)\n", module, len(snapshot.Benchmarks))
	}

	if failed {
		return 1
	}
	return 0
}

func monorepoCheck(ctx context.Context, modules []string, stdout, stderr io.Writer) int {
	failed := false
	for _, module := range modules {
		cfg, err := loadModuleConfig(module)
		if err != nil {
			fmt.Fprintf(stderr, "%s: load config: %v\n", module, err)
			failed = true
			continue
		}

		log := logger.New(io.Discard, slog.LevelWarn)
		runner := gotest.NewRunner(module, log)
		store := storage.NewJSONStorage()
		runBenchmarks := usecases.NewRunBenchmarks(runner, log)
		uc := usecases.NewCheckRegression(runBenchmarks, store, log)
		result, err := uc.Execute(ctx, cfg)
		if err != nil {
			fmt.Fprintln(stderr, formatCommandError(module+" check regressions", err))
			failed = true
			continue
		}

		fmt.Fprintf(stdout, "\n## %s\n", module)
		if err := reporter.NewTerminal().PrintCheck(stdout, result); err != nil {
			fmt.Fprintf(stderr, "%s print report: %v\n", module, err)
			failed = true
			continue
		}
		if result.Failed && cfg.Output.FailOnRegression {
			failed = true
		}
	}

	if failed {
		return 1
	}
	return 0
}

func loadModuleConfig(module string) (usecases.Config, error) {
	cfg, err := loadConfig(filepath.Join(module, "gopulse.yaml"))
	if err != nil {
		return usecases.Config{}, err
	}
	cfg.WorkingDirectory = module
	cfg.BaselinePath = filepath.Join(module, cfg.BaselinePath)
	return cfg, nil
}

func findGoModules(root string) []string {
	var modules []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}

		dir := filepath.Dir(path)
		if dir == "." {
			modules = append(modules, ".")
			return nil
		}
		modules = append(modules, strings.TrimPrefix(dir, "."+string(os.PathSeparator)))
		return nil
	})
	sort.Slice(modules, func(i, j int) bool {
		if modules[i] == "." {
			return true
		}
		if modules[j] == "." {
			return false
		}
		return modules[i] < modules[j]
	})
	return modules
}

func findNestedModules(root string) []string {
	var nested []string
	for _, module := range findGoModules(root) {
		if module != "." {
			nested = append(nested, module)
		}
	}
	return nested
}

func printNestedModuleHint(w io.Writer, modules []string) {
	if len(modules) == 0 {
		return
	}

	fmt.Fprintln(w, "Nested Go modules found. Try one of:")
	limit := len(modules)
	if limit > 5 {
		limit = 5
	}
	for _, module := range modules[:limit] {
		fmt.Fprintf(w, "  cd %s && gopulse doctor\n", module)
	}
	if len(modules) > limit {
		fmt.Fprintf(w, "  ...and %d more\n", len(modules)-limit)
	}
}
