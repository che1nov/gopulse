package reporter

import (
	"fmt"
	"io"

	"github.com/che1nov/gopulse/internal/domain"
)

type Markdown struct{}

func NewMarkdown() Markdown {
	return Markdown{}
}

func (r Markdown) PrintSnapshot(w io.Writer, snapshot domain.Snapshot) error {
	if _, err := fmt.Fprintln(w, "# Benchmark results\n\n| Benchmark | ns/op | B/op | allocs/op |\n|---|---:|---:|---:|"); err != nil {
		return err
	}
	for _, b := range snapshot.Benchmarks {
		if _, err := fmt.Fprintf(w, "| %s | %.0f | %.0f | %.0f |\n", b.Name, b.NsPerOp, b.BytesPerOp, b.AllocsPerOp); err != nil {
			return err
		}
	}
	return nil
}

func (r Markdown) PrintCheck(w io.Writer, result domain.CheckResult) error {
	if _, err := fmt.Fprintln(w, "# Performance report\n\n| Benchmark | Metric | Baseline | Current | Change | Status |\n|---|---|---:|---:|---:|---|"); err != nil {
		return err
	}
	for _, cmp := range result.Comparisons {
		if cmp.Missing {
			if _, err := fmt.Fprintf(w, "| %s | - | - | - | - | missing |\n", cmp.Name); err != nil {
				return err
			}
			continue
		}
		if cmp.New {
			if _, err := fmt.Fprintf(w, "| %s | - | - | - | - | new |\n", cmp.Name); err != nil {
				return err
			}
			continue
		}
		for _, metric := range cmp.Metrics {
			status := "OK"
			if metric.Regression {
				status = "REGRESSION"
			}
			if _, err := fmt.Fprintf(w, "| %s | %s | %.0f | %.0f | %+0.1f%% | %s |\n", cmp.Name, metric.Name, metric.Baseline, metric.Current, metric.ChangePct, status); err != nil {
				return err
			}
		}
	}
	if result.Failed {
		_, err := fmt.Fprintf(w, "\n**Result:** FAILED\n\n**Reason:** %s\n", result.Reason)
		return err
	}
	_, err := fmt.Fprintln(w, "\n**Result:** OK")
	return err
}
