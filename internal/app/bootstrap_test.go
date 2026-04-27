package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	oldVersion := Version
	Version = "v9.9.9"
	t.Cleanup(func() { Version = oldVersion })

	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "gopulse v9.9.9" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestFindGoModules(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/root\n")
	writeFile(t, filepath.Join(root, "users-service", "go.mod"), "module example.com/users\n")
	writeFile(t, filepath.Join(root, "vendor", "ignored", "go.mod"), "module example.com/ignored\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatal(err)
		}
	})
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	modules := findGoModules(".")
	if len(modules) != 2 {
		t.Fatalf("modules = %#v, want 2 modules", modules)
	}
	if modules[0] != "." || modules[1] != "users-service" {
		t.Fatalf("modules = %#v", modules)
	}
}

func TestRunReportsNoPackages(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/empty\n")
	withWorkDir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"run"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1; stdout = %s; stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "no Go packages matched ./...") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestDoctorReportsProjectWithoutBenchmarks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/no_benchmarks\n")
	writeFile(t, filepath.Join(root, "calculator.go"), `package calculator

func Add(a, b int) int {
	return a + b
}
`)
	withWorkDir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1; stdout = %s; stderr = %s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"Packages found: 1", "Benchmarks found: 0"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
	if !strings.Contains(stderr.String(), "project is not ready") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestBaselineSaveAndCheck(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkModule(t, root, "example.com/bench")
	writeFile(t, filepath.Join(root, "gopulse.yaml"), `benchmark:
  packages:
    - ./...
  count: 1
  timeout: 5m
thresholds:
  ns_per_op: 100000
  bytes_per_op: 100000
  allocs_per_op: 100000
output:
  format: terminal
  fail_on_regression: true
`)
	withWorkDir(t, root)

	var saveOut, saveErr bytes.Buffer
	saveCode := Run([]string{"baseline", "save"}, &saveOut, &saveErr)
	if saveCode != 0 {
		t.Fatalf("baseline save code = %d; stdout = %s; stderr = %s", saveCode, saveOut.String(), saveErr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".gopulse", "baseline.json")); err != nil {
		t.Fatalf("baseline file not created: %v", err)
	}

	var checkOut, checkErr bytes.Buffer
	checkCode := Run([]string{"check"}, &checkOut, &checkErr)
	if checkCode != 0 {
		t.Fatalf("check code = %d; stdout = %s; stderr = %s", checkCode, checkOut.String(), checkErr.String())
	}
	if !strings.Contains(checkOut.String(), "Result: OK") {
		t.Fatalf("check stdout = %q", checkOut.String())
	}

	var reportOut, reportErr bytes.Buffer
	reportCode := Run([]string{"report", "--format", "html"}, &reportOut, &reportErr)
	if reportCode != 0 {
		t.Fatalf("html report code = %d; stdout = %s; stderr = %s", reportCode, reportOut.String(), reportErr.String())
	}
	for _, want := range []string{"<!doctype html>", "Performance report", `class="bar"`} {
		if !strings.Contains(reportOut.String(), want) {
			t.Fatalf("html report stdout = %q, want %q", reportOut.String(), want)
		}
	}
}

func TestMonorepoWorkflow(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/root\n")
	writeBenchmarkModule(t, filepath.Join(root, "users-service"), "example.com/users")
	withWorkDir(t, root)

	var doctorOut, doctorErr bytes.Buffer
	doctorCode := Run([]string{"monorepo", "doctor"}, &doctorOut, &doctorErr)
	if doctorCode != 1 {
		t.Fatalf("doctor code = %d, want 1 for root module without packages; stdout = %s; stderr = %s", doctorCode, doctorOut.String(), doctorErr.String())
	}
	for _, want := range []string{"Go modules found: 2", "users-service", "benchmarks: 1"} {
		if !strings.Contains(doctorOut.String(), want) {
			t.Fatalf("doctor stdout = %q, want %q", doctorOut.String(), want)
		}
	}

	var runOut, runErr bytes.Buffer
	runCode := Run([]string{"monorepo", "run"}, &runOut, &runErr)
	if runCode != 1 {
		t.Fatalf("run code = %d, want 1 because root module has no packages; stdout = %s; stderr = %s", runCode, runOut.String(), runErr.String())
	}
	if !strings.Contains(runOut.String(), "BenchmarkAdd") {
		t.Fatalf("run stdout = %q", runOut.String())
	}
}

func withWorkDir(t *testing.T, dir string) {
	t.Helper()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatal(err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
}

func writeBenchmarkModule(t *testing.T, root, module string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "go.mod"), "module "+module+"\n")
	writeFile(t, filepath.Join(root, "calculator.go"), `package calculator

func Add(a, b int) int {
	return a + b
}
`)
	writeFile(t, filepath.Join(root, "calculator_test.go"), `package calculator

import "testing"

func BenchmarkAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Add(i, i)
	}
}
`)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
