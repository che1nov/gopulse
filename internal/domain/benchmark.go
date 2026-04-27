package domain

import "time"

type Benchmark struct {
	Name        string  `json:"name"`
	Package     string  `json:"package"`
	NsPerOp     float64 `json:"ns_per_op"`
	BytesPerOp  float64 `json:"bytes_per_op"`
	AllocsPerOp float64 `json:"allocs_per_op"`
}

type Snapshot struct {
	Project    string      `json:"project"`
	GoVersion  string      `json:"go_version"`
	CreatedAt  time.Time   `json:"created_at"`
	Benchmarks []Benchmark `json:"benchmarks"`
}

func (b Benchmark) Key() string {
	if b.Package == "" {
		return b.Name
	}
	return b.Package + "/" + b.Name
}
