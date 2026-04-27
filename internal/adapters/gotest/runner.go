package gotest

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/che1nov/gopulse/internal/domain"
	"github.com/che1nov/gopulse/internal/usecases"
)

var benchLineRE = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([\d.]+)\s+ns/op(?:\s+([\d.]+)\s+B/op)?(?:\s+([\d.]+)\s+allocs/op)?`)
var benchFuncRE = regexp.MustCompile(`func\s+Benchmark[A-Za-z0-9_]*\s*\(`)

type Runner struct {
	workDir string
	logger  *slog.Logger
}

type NoPackagesError struct {
	Patterns []string
}

func (e NoPackagesError) Error() string {
	return fmt.Sprintf("no Go packages matched %s", strings.Join(e.Patterns, ", "))
}

type NoBenchmarksError struct{}

func (e NoBenchmarksError) Error() string {
	return "no benchmarks found in go test output"
}

func NewRunner(workDir string, logger *slog.Logger) Runner {
	return Runner{workDir: workDir, logger: logger}
}

func (r Runner) Run(ctx context.Context, cfg usecases.BenchmarkConfig) ([]domain.Benchmark, error) {
	if _, err := r.Packages(ctx, cfg.Packages); err != nil {
		return nil, err
	}

	args := []string{"test", "-bench=.", "-benchmem", "-run=^$"}
	if cfg.Count > 0 {
		args = append(args, "-count="+strconv.Itoa(cfg.Count))
	}
	if cfg.Timeout != "" {
		args = append(args, "-timeout="+cfg.Timeout)
	}
	args = append(args, cfg.Packages...)

	r.logger.InfoContext(ctx, "go test started", "operation", "go_test_bench", "args", args)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = r.workDir

	out, err := cmd.CombinedOutput()
	benchmarks, parseErr := ParseBenchmarks(out)
	if err != nil {
		return nil, fmt.Errorf("go test benchmark failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	if parseErr != nil {
		return nil, parseErr
	}
	if len(benchmarks) == 0 {
		return nil, NoBenchmarksError{}
	}
	return benchmarks, nil
}

func (r Runner) Packages(ctx context.Context, patterns []string) ([]string, error) {
	args := append([]string{"list"}, patterns...)
	out, err := r.goStdout(ctx, args...)
	if err != nil {
		text := strings.TrimSpace(string(out))
		if strings.Contains(text, "matched no packages") || strings.Contains(text, "no packages to test") {
			return nil, NoPackagesError{Patterns: patterns}
		}
		return nil, fmt.Errorf("go list failed: %w\n%s", err, text)
	}

	lines := strings.Fields(strings.TrimSpace(string(out)))
	if len(lines) == 0 {
		return nil, NoPackagesError{Patterns: patterns}
	}
	return lines, nil
}

func (r Runner) BenchmarkFileCount(ctx context.Context, patterns []string) (int, error) {
	args := append([]string{"list", "-f", `{{.Dir}}|{{range .TestGoFiles}}{{.}};{{end}}{{range .XTestGoFiles}}{{.}};{{end}}`}, patterns...)
	out, err := r.goStdout(ctx, args...)
	if err != nil {
		text := strings.TrimSpace(string(out))
		if strings.Contains(text, "matched no packages") {
			return 0, nil
		}
		return 0, fmt.Errorf("go list test files failed: %w\n%s", err, text)
	}

	count := 0
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		dir, files, ok := strings.Cut(scanner.Text(), "|")
		if !ok || files == "" {
			continue
		}
		for _, name := range strings.Split(files, ";") {
			if name == "" {
				continue
			}
			count += countBenchmarks(filepath.Join(dir, name))
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

func (r Runner) HasImport(ctx context.Context, patterns []string, importPath string) bool {
	format := `{{range .Imports}}{{if eq . "` + importPath + `"}}yes{{end}}{{end}}{{range .TestImports}}{{if eq . "` + importPath + `"}}yes{{end}}{{end}}{{range .XTestImports}}{{if eq . "` + importPath + `"}}yes{{end}}{{end}}`
	args := append([]string{"list", "-f", format}, patterns...)
	out, err := r.goStdout(ctx, args...)
	return err == nil && strings.Contains(string(out), "yes")
}

func (r Runner) Project(ctx context.Context) string {
	out, err := r.goStdout(ctx, "list", "-m")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (r Runner) GoVersion(ctx context.Context) string {
	out, err := r.goStdout(ctx, "env", "GOVERSION")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (r Runner) goOutput(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = r.workDir
	return cmd.CombinedOutput()
}

func (r Runner) goStdout(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = r.workDir
	return cmd.Output()
}

func ParseBenchmarks(out []byte) ([]domain.Benchmark, error) {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	currentPackage := ""
	var benchmarks []domain.Benchmark

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "pkg: ") {
			currentPackage = strings.TrimSpace(strings.TrimPrefix(line, "pkg: "))
			continue
		}

		matches := benchLineRE.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		ns, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			return nil, fmt.Errorf("parse ns/op: %w", err)
		}

		benchmark := domain.Benchmark{
			Name:    strings.Split(matches[1], "-")[0],
			Package: currentPackage,
			NsPerOp: ns,
		}
		if matches[3] != "" {
			benchmark.BytesPerOp, err = strconv.ParseFloat(matches[3], 64)
			if err != nil {
				return nil, fmt.Errorf("parse B/op: %w", err)
			}
		}
		if matches[4] != "" {
			benchmark.AllocsPerOp, err = strconv.ParseFloat(matches[4], 64)
			if err != nil {
				return nil, fmt.Errorf("parse allocs/op: %w", err)
			}
		}

		benchmarks = append(benchmarks, benchmark)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return averageBenchmarks(benchmarks), nil
}

func averageBenchmarks(benchmarks []domain.Benchmark) []domain.Benchmark {
	type aggregate struct {
		benchmark domain.Benchmark
		count     float64
	}

	byKey := make(map[string]aggregate, len(benchmarks))
	order := make([]string, 0, len(benchmarks))
	for _, b := range benchmarks {
		key := b.Key()
		agg, ok := byKey[key]
		if !ok {
			order = append(order, key)
			agg.benchmark.Name = b.Name
			agg.benchmark.Package = b.Package
		}
		agg.benchmark.NsPerOp += b.NsPerOp
		agg.benchmark.BytesPerOp += b.BytesPerOp
		agg.benchmark.AllocsPerOp += b.AllocsPerOp
		agg.count++
		byKey[key] = agg
	}

	result := make([]domain.Benchmark, 0, len(byKey))
	for _, key := range order {
		agg := byKey[key]
		agg.benchmark.NsPerOp /= agg.count
		agg.benchmark.BytesPerOp /= agg.count
		agg.benchmark.AllocsPerOp /= agg.count
		result = append(result, agg.benchmark)
	}
	return result
}

func containsFile(path, needle string) bool {
	data, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(data), needle)
}

func countBenchmarks(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return len(benchFuncRE.FindAll(data, -1))
}
