package app

import (
	"context"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/che1nov/gopulse/internal/adapters/gotest"
	"github.com/che1nov/gopulse/internal/adapters/reporter"
	"github.com/che1nov/gopulse/internal/adapters/storage"
	"github.com/che1nov/gopulse/internal/domain"
	"github.com/che1nov/gopulse/internal/usecases"
	"github.com/che1nov/gopulse/pkg/logger"
)

func Run(args []string, stdout, stderr io.Writer) int {
	log := logger.New(stderr, slog.LevelWarn)
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	ctx := context.Background()
	cfg, err := loadConfig("gopulse.yaml")
	if err != nil {
		fmt.Fprintf(stderr, "load config: %v\n", err)
		return 1
	}

	runner := gotest.NewRunner(".", log.With("component", "gotest"))
	store := storage.NewJSONStorage()
	runBenchmarks := usecases.NewRunBenchmarks(runner, log.With("component", "usecase"))
	checkRegression := usecases.NewCheckRegression(runBenchmarks, store, log.With("component", "usecase"))

	switch args[0] {
	case "run":
		return run(ctx, args[1:], cfg, runBenchmarks, stdout, stderr)
	case "baseline":
		return baseline(ctx, args[1:], cfg, runBenchmarks, store, log, stdout, stderr)
	case "check":
		return check(ctx, args[1:], cfg, checkRegression, stdout, stderr)
	case "report":
		return report(ctx, args[1:], cfg, checkRegression, stdout, stderr)
	case "doctor":
		return doctor(cfg, store, stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func run(ctx context.Context, args []string, cfg usecases.Config, uc usecases.RunBenchmarks, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", string(cfg.Output.Format), "output format: terminal, markdown, json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	snapshot, err := uc.Execute(ctx, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if err := reporterFor(domain.ReportFormat(*format)).PrintSnapshot(stdout, snapshot); err != nil {
		fmt.Fprintf(stderr, "print report: %v\n", err)
		return 1
	}
	return 0
}

func baseline(ctx context.Context, args []string, cfg usecases.Config, runner usecases.RunBenchmarks, store usecases.BaselineStorage, log *slog.Logger, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "save" {
		fmt.Fprintln(stderr, "usage: gopulse baseline save")
		return 2
	}

	uc := usecases.NewSaveBaseline(runner, store, log.With("component", "usecase"))
	snapshot, err := uc.Execute(ctx, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Baseline saved: %s (%d benchmarks)\n", cfg.BaselinePath, len(snapshot.Benchmarks))
	return 0
}

func check(ctx context.Context, args []string, cfg usecases.Config, uc usecases.CheckRegression, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", string(cfg.Output.Format), "output format: terminal, markdown, json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	result, err := uc.Execute(ctx, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if err := reporterFor(domain.ReportFormat(*format)).PrintCheck(stdout, result); err != nil {
		fmt.Fprintf(stderr, "print report: %v\n", err)
		return 1
	}
	if result.Failed && cfg.Output.FailOnRegression {
		return 1
	}
	return 0
}

func report(ctx context.Context, args []string, cfg usecases.Config, uc usecases.CheckRegression, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "markdown", "output format: markdown, terminal, json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	result, err := uc.Execute(ctx, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if err := reporterFor(domain.ReportFormat(*format)).PrintCheck(stdout, result); err != nil {
		fmt.Fprintf(stderr, "print report: %v\n", err)
		return 1
	}
	return 0
}

func doctor(cfg usecases.Config, store usecases.BaselineStorage, stdout, stderr io.Writer) int {
	moduleOK := fileExists("go.mod")
	benchmarks := countBenchmarkFiles(".")
	pprofFound := hasGoImport(".", "net/http/pprof") || hasGoImport(".", "runtime/pprof")

	fmt.Fprintf(stdout, "Go module: %s\n", status(moduleOK))
	fmt.Fprintf(stdout, "Benchmarks found: %d\n", benchmarks)
	fmt.Fprintf(stdout, "Baseline found: %s\n", status(store.Exists(cfg.BaselinePath)))
	fmt.Fprintln(stdout, "Benchmem enabled: OK")
	fmt.Fprintf(stdout, "CI config: %s\n", status(fileExists(".github/workflows/performance.yml") || fileExists(".github/workflows/performance.yaml")))
	fmt.Fprintf(stdout, "pprof usage: %s\n", status(pprofFound))

	if !moduleOK || benchmarks == 0 {
		fmt.Fprintln(stderr, "project is not ready for performance analysis")
		return 1
	}
	return 0
}

type reportPrinter interface {
	PrintSnapshot(io.Writer, domain.Snapshot) error
	PrintCheck(io.Writer, domain.CheckResult) error
}

func reporterFor(format domain.ReportFormat) reportPrinter {
	switch format {
	case domain.ReportMarkdown:
		return reporter.NewMarkdown()
	case domain.ReportJSON:
		return reporter.NewJSON()
	default:
		return reporter.NewTerminal()
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `gopulse checks performance health of Go projects.

Usage:
  gopulse run [--format terminal|markdown|json]
  gopulse baseline save
  gopulse check [--format terminal|markdown|json]
  gopulse report --format markdown
  gopulse doctor`)
}

func status(ok bool) string {
	if ok {
		return "OK"
	}
	return "not found"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func countBenchmarkFiles(root string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, "_test.go") && containsFile(path, "func Benchmark") {
			count++
		}
		return nil
	})
	return count
}

func hasGoImport(root, importPath string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if found || err != nil || d.IsDir() {
			if d != nil && d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && fileImports(path, importPath) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func fileImports(path, importPath string) bool {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return false
	}
	for _, spec := range file.Imports {
		if strings.Trim(spec.Path.Value, `"`) == importPath {
			return true
		}
	}
	return false
}

func containsFile(path, needle string) bool {
	data, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(data), needle)
}
