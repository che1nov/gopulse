package app

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/che1nov/gopulse/internal/domain"
	"github.com/che1nov/gopulse/internal/usecases"
)

func loadConfig(path string) (usecases.Config, error) {
	cfg := usecases.DefaultConfig()
	cfg.ConfigPath = path

	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") && !strings.HasPrefix(line, "-") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		if strings.HasPrefix(line, "- ") && section == "benchmark.packages" {
			cfg.Benchmark.Packages = append(cfg.Benchmark.Packages, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.Contains(section, ".") {
			section, _, _ = strings.Cut(section, ".")
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		switch section {
		case "benchmark":
			switch key {
			case "packages":
				cfg.Benchmark.Packages = nil
				section = "benchmark.packages"
			case "count":
				count, err := strconv.Atoi(value)
				if err != nil {
					return cfg, err
				}
				cfg.Benchmark.Count = count
			case "timeout":
				cfg.Benchmark.Timeout = value
			}
		case "thresholds":
			threshold, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return cfg, err
			}
			switch key {
			case "ns_per_op":
				cfg.Thresholds.NsPerOp = threshold
			case "bytes_per_op":
				cfg.Thresholds.BytesPerOp = threshold
			case "allocs_per_op":
				cfg.Thresholds.AllocsPerOp = threshold
			}
		case "output":
			switch key {
			case "format":
				cfg.Output.Format = domain.ReportFormat(value)
			case "fail_on_regression":
				fail, err := strconv.ParseBool(value)
				if err != nil {
					return cfg, err
				}
				cfg.Output.FailOnRegression = fail
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return cfg, err
	}
	if len(cfg.Benchmark.Packages) == 0 {
		cfg.Benchmark.Packages = []string{"./..."}
	}
	return cfg, nil
}
