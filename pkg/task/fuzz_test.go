package task

import (
	"encoding/json"
	"testing"
)

func FuzzTaskFingerprintCompute(f *testing.F) {
	f.Add("summarize", int64(100), int64(500), false, "INTERNAL", "us-central1", int64(2000), 0.05, false, false)
	f.Add("", int64(0), int64(0), true, "", "", int64(0), 0.0, true, true)
	f.Add("rag", int64(-100), int64(-500), false, "RESTRICTED", "europe-west1", int64(-1), -1.0, false, false)

	f.Fuzz(func(t *testing.T, cap string, inBytes, outBytes int64, stream bool, dataClass, region string, maxLat int64, maxCost float64, del, structOut bool) {
		fp := NewTaskFingerprint(
			cap,
			inBytes,
			outBytes,
			stream,
			[]string{"toolA", "toolB"},
			dataClass,
			region,
			maxLat,
			maxCost,
			del,
			[]string{"gemini-1.5-pro"},
			structOut,
		)
		if fp == nil || fp.FingerprintID == "" {
			t.Fatal("fingerprint should not be nil or empty")
		}
		data, err := json.Marshal(fp)
		if err != nil {
			t.Fatalf("failed to marshal fingerprint: %v", err)
		}
		var decoded TaskFingerprint
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal fingerprint: %v", err)
		}
	})
}
