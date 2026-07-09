package stamp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteThenCheckClean(t *testing.T) {
	dir := t.TempDir()
	files := []File{
		{Path: "Chart.yaml", Content: []byte("name: demo\n")},
		{Path: "templates/web/deployment.yaml", Content: []byte("kind: Deployment\n")},
	}
	if err := Write(files, dir); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "templates/web/deployment.yaml")); err != nil || string(got) != "kind: Deployment\n" {
		t.Fatalf("file content = %q err=%v", got, err)
	}
	drift, diffs := Check(files, dir)
	if drift {
		t.Errorf("expected no drift, got diffs: %v", diffs)
	}
}

func TestCheckDetectsDrift(t *testing.T) {
	dir := t.TempDir()
	files := []File{{Path: "Chart.yaml", Content: []byte("name: demo\n")}}
	if err := Write(files, dir); err != nil {
		t.Fatal(err)
	}
	changed := []File{{Path: "Chart.yaml", Content: []byte("name: CHANGED\n")}}
	drift, diffs := Check(changed, dir)
	if !drift || len(diffs) == 0 {
		t.Errorf("expected drift detected")
	}
}
