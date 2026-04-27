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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
