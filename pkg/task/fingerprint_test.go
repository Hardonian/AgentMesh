package task

import (
	"testing"
)

func TestTaskFingerprintDeterministicHash(t *testing.T) {
	fp1 := NewTaskFingerprint(
		"financial_analysis",
		4096,
		16384,
		false,
		[]string{"bigquery.read", "chart.render"},
		"CONFIDENTIAL",
		"us-central1",
		5000,
		0.05,
		false,
		[]string{"gemini-1.5-pro"},
		true,
	)

	// Tools in different initial order to test sorting invariance
	fp2 := NewTaskFingerprint(
		"financial_analysis",
		4096,
		16384,
		false,
		[]string{"chart.render", "bigquery.read"},
		"CONFIDENTIAL",
		"us-central1",
		5000,
		0.05,
		false,
		[]string{"gemini-1.5-pro"},
		true,
	)

	if fp1.FingerprintID == "" {
		t.Fatalf("expected non-empty fingerprint ID")
	}
	if fp1.FingerprintID != fp2.FingerprintID {
		t.Fatalf("expected deterministic fingerprint hash, got %s vs %s", fp1.FingerprintID, fp2.FingerprintID)
	}
	if fp1.InputSizeClass != SizeMedium {
		t.Errorf("expected SizeMedium, got %s", fp1.InputSizeClass)
	}
	if fp1.ComplexityClass != ComplexityComplex {
		t.Errorf("expected ComplexityComplex, got %s", fp1.ComplexityClass)
	}
}

func TestClassifyComplexity(t *testing.T) {
	tests := []struct {
		name       string
		tools      int
		lat        int64
		delegation bool
		want       ComplexityClass
	}{
		{"simple", 0, 1000, false, ComplexitySimple},
		{"standard", 1, 2500, false, ComplexityStandard},
		{"complex", 2, 8000, false, ComplexityComplex},
		{"long_chain", 3, 15000, true, ComplexityLongChain},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyComplexity(tc.tools, tc.lat, tc.delegation)
			if got != tc.want {
				t.Errorf("ClassifyComplexity() = %v, want %v", got, tc.want)
			}
		})
	}
}
