package stamp

import (
	"os/exec"
	"testing"
)

func TestValidateOutputSkipsWhenToolsMissing(t *testing.T) {
	// With no tools installed (or installed), ValidateOutput must not panic and
	// must return nil when given a directory it can read. We only assert it does
	// not error spuriously when helm is absent.
	if _, err := exec.LookPath("helm"); err == nil {
		t.Skip("helm present; skipping the missing-tool path")
	}
	if err := ValidateOutput(t.TempDir()); err != nil {
		t.Errorf("expected nil when helm absent, got %v", err)
	}
}
