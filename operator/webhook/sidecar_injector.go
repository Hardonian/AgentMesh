package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// AdmissionReview envelopes Kubernetes admission requests and responses.
type AdmissionReview struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Request    *AdmissionRequest  `json:"request,omitempty"`
	Response   *AdmissionResponse `json:"response,omitempty"`
}

// AdmissionRequest contains information describing the incoming resource modification.
type AdmissionRequest struct {
	UID       string          `json:"uid"`
	Namespace string          `json:"namespace"`
	Operation string          `json:"operation"`
	Object    json.RawMessage `json:"object"`
}

// AdmissionResponse contains the admission decision and mutating JSON patch.
type AdmissionResponse struct {
	UID       string `json:"uid"`
	Allowed   bool   `json:"allowed"`
	Patch     []byte `json:"patch,omitempty"`
	PatchType string `json:"patchType,omitempty"` // "JSONPatch"
	Result    string `json:"result,omitempty"`
}

// JSONPatchOperation defines a single RFC 6902 patch operation.
type JSONPatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

// SidecarConfig configures the proxy sidecar container injection.
type SidecarConfig struct {
	ProxyImage        string   `json:"proxyImage"`
	ControlPlaneURL   string   `json:"controlPlaneUrl"`
	ProxyPort         int      `json:"proxyPort"`
	MetricsPort       int      `json:"metricsPort"`
	IgnoredNamespaces []string `json:"ignoredNamespaces"`
}

// SidecarInjector processes Pod admission requests and mutates annotated pods.
type SidecarInjector struct {
	config *SidecarConfig
}

// NewSidecarInjector constructs a mutating admission injector.
func NewSidecarInjector(cfg *SidecarConfig) *SidecarInjector {
	if cfg == nil {
		cfg = &SidecarConfig{
			ProxyImage:      "ghcr.io/agentmesh/agentmesh-proxy:v2.0.0",
			ControlPlaneURL: "http://agentmesh-controller.agentmesh-system.svc.cluster.local:8080",
			ProxyPort:       8081,
			MetricsPort:     8082,
			IgnoredNamespaces: []string{
				"kube-system",
				"kube-public",
				"kube-node-lease",
			},
		}
	}
	return &SidecarInjector{config: cfg}
}

// MutatePod reviews a pod creation request and injects the agentmesh-proxy sidecar if annotated.
func (s *SidecarInjector) MutatePod(req *AdmissionRequest) (*AdmissionResponse, error) {
	if req == nil {
		return nil, errors.New("admission request is nil")
	}

	resp := &AdmissionResponse{
		UID:     req.UID,
		Allowed: true,
	}

	// 1. Check if namespace is ignored
	for _, ign := range s.config.IgnoredNamespaces {
		if strings.EqualFold(req.Namespace, ign) {
			resp.Result = fmt.Sprintf("namespace %q is ignored from injection", req.Namespace)
			return resp, nil
		}
	}

	// 2. Parse pod metadata
	var pod struct {
		Metadata struct {
			Name        string            `json:"name"`
			Namespace   string            `json:"namespace"`
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
		Spec struct {
			Containers []any `json:"containers"`
		} `json:"spec"`
	}

	if err := json.Unmarshal(req.Object, &pod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pod object: %w", err)
	}

	// 3. Check for opt-in annotation: agentmesh.io/inject: "true"
	injectVal, ok := pod.Metadata.Annotations["agentmesh.io/inject"]
	if !ok || !strings.EqualFold(injectVal, "true") {
		resp.Result = "pod does not have agentmesh.io/inject: true annotation"
		return resp, nil
	}

	// 4. Construct sidecar container specification
	sidecarContainer := map[string]any{
		"name":  "agentmesh-proxy",
		"image": s.config.ProxyImage,
		"env": []map[string]string{
			{"name": "AGENTMESH_CONTROL_PLANE", "value": s.config.ControlPlaneURL},
			{"name": "AGENTMESH_PROXY_PORT", "value": fmt.Sprintf("%d", s.config.ProxyPort)},
			{"name": "AGENTMESH_METRICS_PORT", "value": fmt.Sprintf("%d", s.config.MetricsPort)},
			{"name": "AGENTMESH_POD_NAME", "value": pod.Metadata.Name},
			{"name": "AGENTMESH_NAMESPACE", "value": req.Namespace},
		},
		"ports": []map[string]any{
			{"name": "mesh-proxy", "containerPort": s.config.ProxyPort},
			{"name": "mesh-metrics", "containerPort": s.config.MetricsPort},
		},
		"resources": map[string]any{
			"limits":   map[string]string{"cpu": "500m", "memory": "256Mi"},
			"requests": map[string]string{"cpu": "50m", "memory": "64Mi"},
		},
	}

	patches := []JSONPatchOperation{
		{
			Op:    "add",
			Path:  "/spec/containers/-",
			Value: sidecarContainer,
		},
	}

	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSONPatch: %w", err)
	}

	resp.Patch = patchBytes
	resp.PatchType = "JSONPatch"
	resp.Result = "successfully injected agentmesh-proxy sidecar"
	return resp, nil
}
