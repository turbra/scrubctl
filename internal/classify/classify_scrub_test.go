package classify

import (
	"testing"

	"github.com/turbra/scrubctl/internal/sanitize"
	"github.com/turbra/scrubctl/internal/types"
)

func TestClassifyForDirectScrub(t *testing.T) {
	tests := []struct {
		name               string
		resource           types.ResourceObject
		wantClassification types.Classification
		wantReason         string
		wantSanitizedNil   bool
	}{
		{
			name:               "Pod is review and sanitizes",
			resource:           makeResource("v1", "Pod", "test-pod"),
			wantClassification: types.ClassificationReview,
			wantReason:         "Runtime or cluster-owned resource; review before use",
		},
		{
			name:               "Deployment is include",
			resource:           makeResource("apps/v1", "Deployment", "web"),
			wantClassification: types.ClassificationInclude,
		},
		{
			name:               "Ingress is include",
			resource:           makeResource("networking.k8s.io/v1", "Ingress", "web-ingress"),
			wantClassification: types.ClassificationInclude,
			wantReason:         "Ingress routing resource is declarative",
		},
		{
			name:               "LimitRange is review",
			resource:           makeResource("v1", "LimitRange", "default-limits"),
			wantClassification: types.ClassificationReview,
			wantReason:         "Namespace policy resource may not belong in app manifests",
		},
		{
			name:               "PodDisruptionBudget is review",
			resource:           makeResource("policy/v1", "PodDisruptionBudget", "web-pdb"),
			wantClassification: types.ClassificationReview,
			wantReason:         "Availability policy should be reviewed in context",
		},
		{
			name:               "ResourceQuota is review",
			resource:           makeResource("v1", "ResourceQuota", "ns-quota"),
			wantClassification: types.ClassificationReview,
			wantReason:         "Namespace policy resource may not belong in app manifests",
		},
		{
			name:               "unknown kind rejected",
			resource:           makeResource("example.com/v1", "MyCustomThing", "foo"),
			wantClassification: types.ClassificationExclude,
			wantReason:         "kind not supported for direct scrub",
			wantSanitizedNil:   true,
		},
		{
			name: "ownerReferences is review not exclude",
			resource: types.ResourceObject{
				"apiVersion": "apps/v1",
				"kind":       "ReplicaSet",
				"metadata": map[string]any{
					"name":            "web-abc123",
					"namespace":       "demo",
					"ownerReferences": []any{map[string]any{"kind": "Deployment", "name": "web"}},
				},
			},
			wantClassification: types.ClassificationReview,
			wantReason:         "Controller-owned resource",
		},
		{
			name:               "OpenShift scaffolding ServiceAccount is review not exclude",
			resource:           makeResource("v1", "ServiceAccount", "default"),
			wantClassification: types.ClassificationReview,
			wantReason:         "OpenShift-injected namespace scaffolding",
		},
		{
			name:               "OpenShift scaffolding ConfigMap is review not exclude",
			resource:           makeResource("v1", "ConfigMap", "kube-root-ca.crt"),
			wantClassification: types.ClassificationReview,
			wantReason:         "OpenShift-injected namespace scaffolding",
		},
		{
			name:               "ClusterRole allowed in broad set",
			resource:           makeResource("rbac.authorization.k8s.io/v1", "ClusterRole", "admin"),
			wantClassification: types.ClassificationInclude,
		},
		{
			name:               "Namespace allowed in broad set",
			resource:           makeResource("v1", "Namespace", "my-ns"),
			wantClassification: types.ClassificationInclude,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyForDirectScrub(tt.resource)
			if result.Classification != tt.wantClassification {
				t.Errorf("classification = %q, want %q", result.Classification, tt.wantClassification)
			}
			if tt.wantReason != "" && result.Reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", result.Reason, tt.wantReason)
			}

			sanitized := sanitize.SanitizeResource(tt.resource, result.Classification, "redact")
			if tt.wantSanitizedNil {
				if sanitized != nil {
					t.Error("expected nil sanitized output for excluded resource")
				}
			} else {
				if sanitized == nil {
					t.Error("expected non-nil sanitized output; review/include resources must produce output")
				}
			}
		})
	}
}

func makeResource(apiVersion, kind, name string) types.ResourceObject {
	return types.ResourceObject{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata": map[string]any{
			"name":      name,
			"namespace": "demo",
		},
	}
}
