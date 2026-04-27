package usecases

import "github.com/che1nov/gopulse/internal/domain"

type Config struct {
	Benchmark        BenchmarkConfig
	Thresholds       domain.Thresholds
	Output           OutputConfig
	BaselinePath     string
	ConfigPath       string
	WorkingDirectory string
}

type BenchmarkConfig struct {
	Packages []string
	Count    int
	Timeout  string
}

type OutputConfig struct {
	Format           domain.ReportFormat
	FailOnRegression bool
}

func DefaultConfig() Config {
	return Config{
		Benchmark: BenchmarkConfig{
			Packages: []string{"./..."},
			Count:    1,
			Timeout:  "5m",
		},
		Thresholds: domain.DefaultThresholds(),
		Output: OutputConfig{
			Format:           domain.ReportTerminal,
			FailOnRegression: true,
		},
		BaselinePath: ".gopulse/baseline.json",
		ConfigPath:   "gopulse.yaml",
	}
}
