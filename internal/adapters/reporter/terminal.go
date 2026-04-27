package reporter

import (
	"fmt"
	"io"

	"github.com/che1nov/gopulse/internal/domain"
)

type Terminal struct{}

func NewTerminal() Terminal {
	return Terminal{}
}

func (r Terminal) PrintSnapshot(w io.Writer, snapshot domain.Snapshot) error {
	if _, err := fmt.Fprintln(w, "Benchmark results"); err != nil {
		return err
	}
	for _, b := range snapshot.Benchmarks {
		if _, err := fmt.Fprintf(w, "%s\n  ns/op: %.0f\n  B/op: %.0f\n  allocs/op: %.0f\n", b.Name, b.NsPerOp, b.BytesPerOp, b.AllocsPerOp); err != nil {
			return err
		}
	}
	return nil
}

func (r Terminal) PrintCheck(w io.Writer, result domain.CheckResult) error {
	if _, err := fmt.Fprintln(w, "Performance report"); err != nil {
		return err
	}
	for _, cmp := range result.Comparisons {
		if cmp.Missing {
			if _, err := fmt.Fprintf(w, "%s\n  missing in current run\n", cmp.Name); err != nil {
				return err
			}
			continue
		}
		if cmp.New {
			if _, err := fmt.Fprintf(w, "%s\n  new benchmark\n", cmp.Name); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintln(w, cmp.Name); err != nil {
			return err
		}
		for _, metric := range cmp.Metrics {
			status := "OK"
			if metric.Regression {
				status = "REGRESSION"
			}
			if _, err := fmt.Fprintf(w, "  %-9s %.0f -> %.0f   %+0.1f%%   %s\n", metric.Name+":", metric.Baseline, metric.Current, metric.ChangePct, status); err != nil {
				return err
			}
		}
	}

	if result.Failed {
		_, err := fmt.Fprintf(w, "Result: FAILED\nReason: %s\n", result.Reason)
		return err
	}
	_, err := fmt.Fprintln(w, "Result: OK")
	return err
}
