package domain

import "math"

type Thresholds struct {
	NsPerOp     float64 `json:"ns_per_op"`
	BytesPerOp  float64 `json:"bytes_per_op"`
	AllocsPerOp float64 `json:"allocs_per_op"`
}

func DefaultThresholds() Thresholds {
	return Thresholds{
		NsPerOp:     15,
		BytesPerOp:  20,
		AllocsPerOp: 10,
	}
}

type MetricComparison struct {
	Name       string
	Baseline   float64
	Current    float64
	ChangePct  float64
	Threshold  float64
	Regression bool
}

type BenchmarkComparison struct {
	Name    string
	Package string
	Metrics []MetricComparison
	Missing bool
	New     bool
}

type CheckResult struct {
	Comparisons []BenchmarkComparison
	Failed      bool
	Reason      string
}

func CompareSnapshots(baseline, current Snapshot, thresholds Thresholds) CheckResult {
	currentByKey := make(map[string]Benchmark, len(current.Benchmarks))
	for _, b := range current.Benchmarks {
		currentByKey[b.Key()] = b
	}

	var result CheckResult
	for _, base := range baseline.Benchmarks {
		cur, ok := currentByKey[base.Key()]
		if !ok {
			result.Comparisons = append(result.Comparisons, BenchmarkComparison{
				Name:    base.Name,
				Package: base.Package,
				Missing: true,
			})
			continue
		}

		cmp := BenchmarkComparison{
			Name:    base.Name,
			Package: base.Package,
			Metrics: []MetricComparison{
				compareMetric("ns/op", base.NsPerOp, cur.NsPerOp, thresholds.NsPerOp),
				compareMetric("B/op", base.BytesPerOp, cur.BytesPerOp, thresholds.BytesPerOp),
				compareMetric("allocs/op", base.AllocsPerOp, cur.AllocsPerOp, thresholds.AllocsPerOp),
			},
		}
		for _, metric := range cmp.Metrics {
			if metric.Regression {
				result.Failed = true
			}
		}
		result.Comparisons = append(result.Comparisons, cmp)
		delete(currentByKey, base.Key())
	}

	for _, cur := range currentByKey {
		result.Comparisons = append(result.Comparisons, BenchmarkComparison{
			Name:    cur.Name,
			Package: cur.Package,
			New:     true,
		})
	}

	if result.Failed {
		result.Reason = "performance regression above threshold"
	}
	return result
}

func compareMetric(name string, baseline, current, threshold float64) MetricComparison {
	change := 0.0
	if baseline != 0 {
		change = ((current - baseline) / baseline) * 100
	}

	return MetricComparison{
		Name:       name,
		Baseline:   baseline,
		Current:    current,
		ChangePct:  round(change, 1),
		Threshold:  threshold,
		Regression: baseline > 0 && change > threshold,
	}
}

func round(v float64, precision int) float64 {
	pow := math.Pow(10, float64(precision))
	return math.Round(v*pow) / pow
}
