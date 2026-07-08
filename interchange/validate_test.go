package interchange

import (
	"os"
	"strings"
	"testing"
)

func TestValidateAcceptsMinimal(t *testing.T) {
	data, _ := os.ReadFile("../testdata/minimal.json")
	if err := Validate(data); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateRejectsMissingChart(t *testing.T) {
	data, _ := os.ReadFile("../testdata/invalid-missing-chart.json")
	err := Validate(data)
	if err == nil {
		t.Fatal("expected error for missing chart")
	}
	if !strings.Contains(err.Error(), "chart") {
		t.Errorf("error should mention chart, got: %v", err)
	}
}
