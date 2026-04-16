package openshift

import "github.com/turbra/scrubctl/internal/types"

func IsScaffoldingResource(resource types.ResourceObject) bool {
	switch resource.Kind() {
	case "ConfigMap":
		return isInjectedConfigMap(resource.Name())
	case "ServiceAccount":
		return resource.Name() == "default" || resource.Name() == "builder" || resource.Name() == "deployer" || resource.Name() == "pipeline"
	case "RoleBinding":
		switch resource.Name() {
		case "system:deployers", "system:image-builders", "system:image-pullers", "openshift-pipelines-edit", "pipelines-scc-rolebinding":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func isInjectedConfigMap(name string) bool {
	return name == "kube-root-ca.crt" ||
		name == "openshift-service-ca.crt" ||
		contains(name, "service-cabundle") ||
		contains(name, "trusted-cabundle") ||
		contains(name, "ca-bundle")
}

func contains(s, part string) bool {
	return len(s) >= len(part) && (s == part || index(s, part) >= 0)
}

func index(s, sep string) int {
	n := len(sep)
	if n == 0 {
		return 0
	}
	for i := 0; i+n <= len(s); i++ {
		if s[i:i+n] == sep {
			return i
		}
	}
	return -1
}
