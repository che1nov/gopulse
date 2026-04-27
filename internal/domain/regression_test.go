package domain

import "testing"

func TestCompareSnapshotsDetectsRegression(t *testing.T) {
	baseline := Snapshot{Benchmarks: []Benchmark{{
		Name:        "BenchmarkCreateUser",
		Package:     "github.com/che1nov/demo/internal/user",
		NsPerOp:     820,
		BytesPerOp:  256,
		AllocsPerOp: 2,
	}}}
	current := Snapshot{Benchmarks: []Benchmark{{
		Name:        "BenchmarkCreateUser",
		Package:     "github.com/che1nov/demo/internal/user",
		NsPerOp:     1040,
		BytesPerOp:  512,
		AllocsPerOp: 5,
	}}}

	result := CompareSnapshots(baseline, current, Thresholds{
		NsPerOp:     15,
		BytesPerOp:  20,
		AllocsPerOp: 10,
	})

	if !result.Failed {
		t.Fatal("expected failed result")
	}
	if got := result.Comparisons[0].Metrics[0].ChangePct; got != 26.8 {
		t.Fatalf("ns/op change = %v, want 26.8", got)
	}
	for _, metric := range result.Comparisons[0].Metrics {
		if !metric.Regression {
			t.Fatalf("metric %s is not marked as regression", metric.Name)
		}
	}
}

func TestCompareSnapshotsIgnoresImprovements(t *testing.T) {
	baseline := Snapshot{Benchmarks: []Benchmark{{
		Name: "BenchmarkFastPath", NsPerOp: 100, BytesPerOp: 100, AllocsPerOp: 10,
	}}}
	current := Snapshot{Benchmarks: []Benchmark{{
		Name: "BenchmarkFastPath", NsPerOp: 80, BytesPerOp: 50, AllocsPerOp: 2,
	}}}

	result := CompareSnapshots(baseline, current, DefaultThresholds())
	if result.Failed {
		t.Fatal("improvement must not fail check")
	}
}
