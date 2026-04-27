package gotest

import "testing"

func TestParseBenchmarks(t *testing.T) {
	out := []byte(`goos: darwin
goarch: arm64
pkg: github.com/che1nov/demo/internal/parser
BenchmarkParseJSON-8   	 1000000	      1250 ns/op	     512 B/op	       3 allocs/op
BenchmarkEncodeJSON-8   	 2000000	       820.5 ns/op	     128 B/op	       1 allocs/op
PASS
`)

	benchmarks, err := ParseBenchmarks(out)
	if err != nil {
		t.Fatalf("ParseBenchmarks() error = %v", err)
	}
	if len(benchmarks) != 2 {
		t.Fatalf("len = %d, want 2", len(benchmarks))
	}

	first := benchmarks[0]
	if first.Name != "BenchmarkParseJSON" {
		t.Fatalf("name = %q, want BenchmarkParseJSON", first.Name)
	}
	if first.Package != "github.com/che1nov/demo/internal/parser" {
		t.Fatalf("package = %q", first.Package)
	}
	if first.NsPerOp != 1250 || first.BytesPerOp != 512 || first.AllocsPerOp != 3 {
		t.Fatalf("metrics = %+v", first)
	}
}

func TestParseBenchmarksAveragesRepeatedRuns(t *testing.T) {
	out := []byte(`pkg: github.com/che1nov/demo/internal/parser
BenchmarkParseJSON-8   	 1000000	      1000 ns/op	     100 B/op	       1 allocs/op
BenchmarkParseJSON-8   	 1000000	      2000 ns/op	     300 B/op	       3 allocs/op
`)

	benchmarks, err := ParseBenchmarks(out)
	if err != nil {
		t.Fatalf("ParseBenchmarks() error = %v", err)
	}
	if len(benchmarks) != 1 {
		t.Fatalf("len = %d, want 1", len(benchmarks))
	}
	got := benchmarks[0]
	if got.NsPerOp != 1500 || got.BytesPerOp != 200 || got.AllocsPerOp != 2 {
		t.Fatalf("averaged metrics = %+v", got)
	}
}

func BenchmarkParseBenchmarks(b *testing.B) {
	out := []byte(`pkg: github.com/che1nov/demo/internal/parser
BenchmarkParseJSON-8   	 1000000	      1250 ns/op	     512 B/op	       3 allocs/op
BenchmarkEncodeJSON-8   	 2000000	       820 ns/op	     128 B/op	       1 allocs/op
`)

	for i := 0; i < b.N; i++ {
		if _, err := ParseBenchmarks(out); err != nil {
			b.Fatal(err)
		}
	}
}
