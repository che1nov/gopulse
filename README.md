# gopulse

`gopulse` is a small CLI for checking performance health of Go projects. It runs Go benchmarks with `go test -bench=. -benchmem`, stores a baseline, compares future runs, and fails CI when speed, memory, or allocation regressions exceed configured thresholds.

## Install

```bash
go install github.com/che1nov/gopulse/cmd/gopulse@latest
```

For local development:

```bash
go run ./cmd/gopulse --help
```

## Commands

```bash
gopulse run
gopulse baseline save
gopulse check
gopulse report --format markdown
gopulse doctor
```

`gopulse check` exits with code `1` when a regression is above the configured threshold.

## Try It

This repository includes a small benchmark target:

```bash
cd examples/demo-service
gopulse doctor
gopulse run
gopulse baseline save
gopulse check
```

## Example Output

```text
Performance report
BenchmarkCreateUser
  ns/op:     820 -> 1040   +26.8%   REGRESSION
  B/op:      256 -> 512    +100.0%  REGRESSION
  allocs/op: 2 -> 5        +150.0%  REGRESSION
Result: FAILED
Reason: performance regression above threshold
```

## Config

Create `gopulse.yaml` in the project root:

```yaml
benchmark:
  packages:
    - ./...
  count: 5
  timeout: 5m
thresholds:
  ns_per_op: 15
  bytes_per_op: 20
  allocs_per_op: 10
output:
  format: terminal
  fail_on_regression: true
```

## Baseline

```bash
gopulse baseline save
```

Creates `.gopulse/baseline.json`:

```json
{
  "project": "github.com/user/project",
  "go_version": "go1.24.0",
  "created_at": "2026-04-27T10:00:00Z",
  "benchmarks": [
    {
      "name": "BenchmarkParseJSON",
      "package": "github.com/user/project/internal/parser",
      "ns_per_op": 1250,
      "bytes_per_op": 512,
      "allocs_per_op": 3
    }
  ]
}
```

## GitHub Actions

```yaml
name: performance
on:
  pull_request:
jobs:
  gopulse:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: Run performance check
        run: |
          go install github.com/che1nov/gopulse/cmd/gopulse@latest
          gopulse check
```

## Roadmap

- `0.1`: run benchmarks and print terminal report
- `0.2`: save baseline and check regressions
- `0.3`: thresholds from `gopulse.yaml` and CI exit code
- `0.4`: Markdown/JSON reports and GitHub Actions docs
- `0.5`: doctor checks and pprof hints
