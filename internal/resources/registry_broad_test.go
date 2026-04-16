package resources

import "testing"

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
