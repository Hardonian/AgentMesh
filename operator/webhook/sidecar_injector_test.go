package webhook_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/agentmesh/agentmesh/operator/webhook"
)

func TestSidecarInjector_AnnotatedPodMutated(t *testing.T) {
	injector := webhook.NewSidecarInjector(nil)

	podJSON := `{
		"metadata": {
			"name": "finance-agent-pod",
			"namespace": "default",
			"annotations": {
				"agentmesh.io/inject": "true"
			}
		},
		"spec": {
			"containers": [
				{"name": "app", "image": "my-agent:v1"}
			]
		}
	}`

	req := &webhook.AdmissionRequest{
		UID:       "adm-uid-1234",
		Namespace: "default",
		Operation: "CREATE",
		Object:    json.RawMessage(podJSON),
	}

	resp, err := injector.MutatePod(req)
	if err != nil {
		t.Fatalf("MutatePod failed: %v", err)
	}

	if !resp.Allowed {
		t.Fatal("expected request to be allowed")
	}
	if len(resp.Patch) == 0 {
		t.Fatal("expected non-empty JSONPatch for annotated pod")
	}
	if !strings.Contains(string(resp.Patch), "agentmesh-proxy") {
		t.Fatalf("expected patch to contain agentmesh-proxy, got: %s", string(resp.Patch))
	}
}

func TestSidecarInjector_UnannotatedPodNotMutated(t *testing.T) {
	injector := webhook.NewSidecarInjector(nil)

	podJSON := `{
		"metadata": {
			"name": "regular-pod",
			"namespace": "default"
		},
		"spec": {
			"containers": [{"name": "app", "image": "nginx"}]
		}
	}`

	req := &webhook.AdmissionRequest{
		UID:       "adm-uid-5678",
		Namespace: "default",
		Operation: "CREATE",
		Object:    json.RawMessage(podJSON),
	}

	resp, err := injector.MutatePod(req)
	if err != nil {
		t.Fatalf("MutatePod failed: %v", err)
	}

	if len(resp.Patch) != 0 {
		t.Fatalf("expected no patch for unannotated pod, got: %s", string(resp.Patch))
	}
}

func TestSidecarInjector_IgnoredNamespaceSkipped(t *testing.T) {
	injector := webhook.NewSidecarInjector(nil)

	podJSON := `{
		"metadata": {
			"name": "coredns",
			"namespace": "kube-system",
			"annotations": {"agentmesh.io/inject": "true"}
		}
	}`

	req := &webhook.AdmissionRequest{
		UID:       "adm-uid-system",
		Namespace: "kube-system",
		Operation: "CREATE",
		Object:    json.RawMessage(podJSON),
	}

	resp, err := injector.MutatePod(req)
	if err != nil {
		t.Fatalf("MutatePod failed: %v", err)
	}

	if len(resp.Patch) != 0 {
		t.Fatal("kube-system pods must never receive sidecar injection")
	}
}
