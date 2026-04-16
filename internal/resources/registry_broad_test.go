package resources

import "testing"

func TestIsBroadScrubKind(t *testing.T) {
	expected := []string{
		"APIService", "ClusterRole", "ClusterRoleBinding", "ConfigMap",
		"CronJob", "CustomResourceDefinition", "DaemonSet", "Deployment",
		"Endpoints", "HorizontalPodAutoscaler", "Ingress", "Job",
		"LimitRange", "MutatingWebhookConfiguration", "Namespace",
		"NetworkPolicy", "PersistentVolume", "PersistentVolumeClaim",
		"Pod", "PodDisruptionBudget", "PriorityClass", "ReplicaSet",
		"ReplicationController", "ResourceQuota", "Role", "RoleBinding",
		"Secret", "Service", "ServiceAccount", "StatefulSet",
		"StorageClass", "ValidatingWebhookConfiguration",
	}
	for _, kind := range expected {
		if !IsBroadScrubKind(kind) {
			t.Errorf("IsBroadScrubKind(%q) = false, want true", kind)
		}
	}
}

func TestIsBroadScrubKind_Unknown(t *testing.T) {
	unknowns := []string{"MyCustomThing", "Widget", ""}
	for _, kind := range unknowns {
		if IsBroadScrubKind(kind) {
			t.Errorf("IsBroadScrubKind(%q) = true, want false", kind)
		}
	}
}

func TestNewCuratedKinds(t *testing.T) {
	newKinds := []struct {
		kind string
		key  string
	}{
		{"Ingress", "networking.k8s.io/v1/Ingress"},
		{"LimitRange", "v1/LimitRange"},
		{"PodDisruptionBudget", "policy/v1/PodDisruptionBudget"},
		{"ResourceQuota", "v1/ResourceQuota"},
	}
	for _, nk := range newKinds {
		t.Run(nk.kind, func(t *testing.T) {
			option, ok := FindByKindName(nk.kind)
			if !ok {
				t.Fatalf("FindByKindName(%q) not found", nk.kind)
			}
			if option.Key != nk.key {
				t.Errorf("Key = %q, want %q", option.Key, nk.key)
			}
		})
	}
}

func TestIngressSelectedByDefault(t *testing.T) {
	option, ok := FindByKindName("Ingress")
	if !ok {
		t.Fatal("Ingress not found in curated set")
	}
	if !option.SelectedByDefault {
		t.Error("Ingress should be SelectedByDefault")
	}
}

func TestPolicyKindsNotSelectedByDefault(t *testing.T) {
	for _, kind := range []string{"LimitRange", "PodDisruptionBudget", "ResourceQuota"} {
		option, ok := FindByKindName(kind)
		if !ok {
			t.Fatalf("%s not found in curated set", kind)
		}
		if option.SelectedByDefault {
			t.Errorf("%s should NOT be SelectedByDefault", kind)
		}
	}
}
