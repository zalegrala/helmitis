package stamp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalegrala/chartwright/interchange"
)

// TestInstallableChart is the installability acceptance gate. It builds the
// realistic fixture into a chart and runs the tiered tool gate against it:
//
//	helm lint  →  helm template  →  kubeconform -strict
//
// This is what proves the *output artifact* is installable, not just that the
// stamping machinery runs. It skips gracefully when helm/kubeconform are absent
// (minimal dev environments) but runs wherever the tools are present (CI).
func TestInstallableChart(t *testing.T) {
	helm, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("helm not on PATH; skipping installability gate")
	}

	data, err := os.ReadFile("../testdata/installable.json")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := interchange.DecodeAndValidate(data)
	if err != nil {
		t.Fatal(err)
	}
	files, err := Build(doc)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := Write(files, dir); err != nil {
		t.Fatal(err)
	}

	// 1. helm lint — non-zero exit only on ERROR (INFO/WARNING are fine).
	if out, err := exec.Command(helm, "lint", dir).CombinedOutput(); err != nil {
		t.Fatalf("helm lint failed:\n%s", out)
	}

	// 2. helm template — must render without error.
	tmpl := exec.Command(helm, "template", "acceptance", dir)
	rendered, err := tmpl.Output()
	if err != nil {
		msg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			msg = string(ee.Stderr)
		}
		t.Fatalf("helm template failed: %s", msg)
	}
	if !strings.Contains(string(rendered), "kind: Deployment") ||
		!strings.Contains(string(rendered), "kind: Service") {
		t.Fatalf("rendered output missing expected objects:\n%s", rendered)
	}

	// 3. kubeconform -strict — validates rendered objects against k8s schemas.
	kube, err := exec.LookPath("kubeconform")
	if err != nil {
		t.Skip("kubeconform not on PATH; validated lint+template only")
	}
	renderedPath := filepath.Join(dir, "rendered.yaml")
	if err := os.WriteFile(renderedPath, rendered, 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(kube, append(kubeconformArgs, renderedPath)...).CombinedOutput(); err != nil {
		t.Fatalf("kubeconform rejected the rendered chart:\n%s", out)
	}
}
