package stamp

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/zalegrala/chartwright/interchange"
)

// TestJsonnetExamplesInstallable exercises the full producer→consumer pipeline
// for each committed example: jsonnet entrypoint → interchange (with inline hole
// markers) → lower → build → installable chart. End-to-end proof that the
// authoring layer produces charts that pass the installability gate. Skips when
// tools are absent.
func TestJsonnetExamplesInstallable(t *testing.T) {
	examples := []string{
		"../examples/minimal/main.jsonnet",
		"../examples/web/main.jsonnet",
		"../examples/version-gating/main.jsonnet",
	}
	for _, ex := range examples {
		t.Run(ex, func(t *testing.T) { assertJsonnetInstallable(t, ex) })
	}
}

func assertJsonnetInstallable(t *testing.T, entrypoint string) {
	t.Helper()
	jsonnet, err := exec.LookPath("jsonnet")
	if err != nil {
		t.Skip("jsonnet not on PATH; skipping jsonnet pipeline gate")
	}
	helm, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("helm not on PATH; skipping installability gate")
	}

	interchangeJSON, err := exec.Command(jsonnet, entrypoint).Output()
	if err != nil {
		msg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			msg = string(ee.Stderr)
		}
		t.Fatalf("jsonnet failed: %s", msg)
	}

	doc, err := interchange.DecodeAndValidate(interchangeJSON)
	if err != nil {
		t.Fatalf("emitted interchange is invalid: %v", err)
	}
	files, err := Build(doc)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := Write(files, dir); err != nil {
		t.Fatal(err)
	}

	if out, err := exec.Command(helm, "lint", dir).CombinedOutput(); err != nil {
		t.Fatalf("helm lint failed:\n%s", out)
	}
	rendered, err := exec.Command(helm, "template", "acc", dir).Output()
	if err != nil {
		t.Fatalf("helm template failed: %v", err)
	}

	kube, err := exec.LookPath("kubeconform")
	if err != nil {
		t.Skip("kubeconform not on PATH; validated lint+template only")
	}
	kc := exec.Command(kube, append(kubeconformArgs, "-")...)
	kc.Stdin = strings.NewReader(string(rendered))
	if out, err := kc.CombinedOutput(); err != nil {
		t.Fatalf("kubeconform rejected the jsonnet-produced chart:\n%s", out)
	}
}
