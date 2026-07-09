package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVersionString(t *testing.T) {
	if version == "" {
		t.Fatal("version must not be empty")
	}
}

func TestRunWritesChart(t *testing.T) {
	out := t.TempDir()
	err := run([]string{"--in", "../../testdata/minimal.json", "--out", out, "--no-validate-output"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "Chart.yaml")); err != nil {
		t.Errorf("Chart.yaml not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "templates/web/deployment.yaml")); err != nil {
		t.Errorf("deployment not written: %v", err)
	}
}

func TestRunCheckCleanThenDrift(t *testing.T) {
	out := t.TempDir()
	if err := run([]string{"--in", "../../testdata/minimal.json", "--out", out, "--no-validate-output"}); err != nil {
		t.Fatal(err)
	}
	// Second run in --check mode should report no drift (exit nil).
	if err := run([]string{"--in", "../../testdata/minimal.json", "--out", out, "--check", "--no-validate-output"}); err != nil {
		t.Errorf("expected clean check, got %v", err)
	}
}
