package stamp

import (
	"fmt"
	"os/exec"
	"strings"
)

// ValidateOutput runs helm lint (and kubeconform if available) against the
// rendered chart directory. Missing tools are skipped with no error, so the
// stamper works in minimal environments; present tools that report problems
// cause a non-nil error.
func ValidateOutput(chartDir string) error {
	if path, err := exec.LookPath("helm"); err == nil {
		cmd := exec.Command(path, "lint", chartDir)
		var outBuf strings.Builder
		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("helm lint failed:\n%s", outBuf.String())
		}
	}
	// kubeconform validates rendered manifests; only run if both helm and
	// kubeconform exist (we need `helm template` to render first).
	helm, helmErr := exec.LookPath("helm")
	kube, kubeErr := exec.LookPath("kubeconform")
	if helmErr == nil && kubeErr == nil {
		tmpl := exec.Command(helm, "template", chartDir)
		rendered, err := tmpl.Output()
		if err != nil {
			return fmt.Errorf("helm template failed: %w", err)
		}
		kc := exec.Command(kube, "-strict", "-summary", "-")
		kc.Stdin = strings.NewReader(string(rendered))
		var kbuf strings.Builder
		kc.Stdout = &kbuf
		kc.Stderr = &kbuf
		if err := kc.Run(); err != nil {
			return fmt.Errorf("kubeconform failed:\n%s", kbuf.String())
		}
	}
	return nil
}
