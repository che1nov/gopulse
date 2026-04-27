package gotest

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/che1nov/gopulse/internal/domain"
	"github.com/che1nov/gopulse/internal/usecases"
)

var benchLineRE = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([\d.]+)\s+ns/op(?:\s+([\d.]+)\s+B/op)?(?:\s+([\d.]+)\s+allocs/op)?`)

type Runner struct {
	workDir string
	logger  *slog.Logger
}

func NewRunner(workDir string, logger *slog.Logger) Runner {
	return Runner{workDir: workDir, logger: logger}
}

func (r Runner) Run(ctx context.Context, cfg usecases.BenchmarkConfig) ([]domain.Benchmark, error) {
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
		return nil, errors.New("no benchmarks found in go test output")
	}
	return benchmarks, nil
}

func (r Runner) Project(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "go", "list", "-m")
	cmd.Dir = r.workDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (r Runner) GoVersion(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "go", "env", "GOVERSION")
	cmd.Dir = r.workDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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
