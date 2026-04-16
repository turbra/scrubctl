package resources

import (
	"strings"

	"github.com/turbra/scrubctl/internal/types"
)

const (
	routeOpenShiftGroup = "route.openshift.io"
	buildOpenShiftGroup = "build.openshift.io"
	imageOpenShiftGroup = "image.openshift.io"
	openShiftVersion    = "v1"
)

type ResourceTypeOption struct {
	Key               string `json:"key" yaml:"key"`
	Label             string `json:"label" yaml:"label"`
	Group             string `json:"group,omitempty" yaml:"group,omitempty"`
	Version           string `json:"version,omitempty" yaml:"version,omitempty"`
	Kind              string `json:"kind" yaml:"kind"`
	Plural            string `json:"plural" yaml:"plural"`
	SelectedByDefault bool   `json:"selectedByDefault,omitempty" yaml:"selectedByDefault,omitempty"`
}

var ResourceTypeOptions = []ResourceTypeOption{
	{Key: "apps/v1/Deployment", Label: "Deployment", Group: "apps", Version: "v1", Kind: "Deployment", Plural: "deployments", SelectedByDefault: true},
	{Key: "apps/v1/StatefulSet", Label: "StatefulSet", Group: "apps", Version: "v1", Kind: "StatefulSet", Plural: "statefulsets", SelectedByDefault: true},
	{Key: "apps/v1/DaemonSet", Label: "DaemonSet", Group: "apps", Version: "v1", Kind: "DaemonSet", Plural: "daemonsets", SelectedByDefault: true},
	{Key: "batch/v1/Job", Label: "Job", Group: "batch", Version: "v1", Kind: "Job", Plural: "jobs", SelectedByDefault: true},
	{Key: "batch/v1/CronJob", Label: "CronJob", Group: "batch", Version: "v1", Kind: "CronJob", Plural: "cronjobs", SelectedByDefault: true},
	{Key: "v1/Service", Label: "Service", Version: "v1", Kind: "Service", Plural: "services", SelectedByDefault: true},
	{Key: routeOpenShiftGroup + "/" + openShiftVersion + "/Route", Label: "Route", Group: routeOpenShiftGroup, Version: openShiftVersion, Kind: "Route", Plural: "routes", SelectedByDefault: true},
	{Key: "v1/Secret", Label: "Secret", Version: "v1", Kind: "Secret", Plural: "secrets", SelectedByDefault: true},
	{Key: "v1/ConfigMap", Label: "ConfigMap", Version: "v1", Kind: "ConfigMap", Plural: "configmaps", SelectedByDefault: true},
	{Key: "v1/PersistentVolumeClaim", Label: "PersistentVolumeClaim", Version: "v1", Kind: "PersistentVolumeClaim", Plural: "persistentvolumeclaims", SelectedByDefault: true},
	{Key: "networking.k8s.io/v1/NetworkPolicy", Label: "NetworkPolicy", Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy", Plural: "networkpolicies", SelectedByDefault: true},
	{Key: "autoscaling/v2/HorizontalPodAutoscaler", Label: "HorizontalPodAutoscaler", Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler", Plural: "horizontalpodautoscalers", SelectedByDefault: true},
	{Key: buildOpenShiftGroup + "/" + openShiftVersion + "/BuildConfig", Label: "BuildConfig", Group: buildOpenShiftGroup, Version: openShiftVersion, Kind: "BuildConfig", Plural: "buildconfigs", SelectedByDefault: true},
	{Key: imageOpenShiftGroup + "/" + openShiftVersion + "/ImageStream", Label: "ImageStream", Group: imageOpenShiftGroup, Version: openShiftVersion, Kind: "ImageStream", Plural: "imagestreams", SelectedByDefault: true},
	{Key: imageOpenShiftGroup + "/" + openShiftVersion + "/ImageStreamTag", Label: "ImageStreamTag", Group: imageOpenShiftGroup, Version: openShiftVersion, Kind: "ImageStreamTag", Plural: "imagestreamtags"},
	{Key: "rbac.authorization.k8s.io/v1/Role", Label: "Role", Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role", Plural: "roles"},
	{Key: "rbac.authorization.k8s.io/v1/RoleBinding", Label: "RoleBinding", Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding", Plural: "rolebindings"},
	{Key: "v1/ServiceAccount", Label: "ServiceAccount", Version: "v1", Kind: "ServiceAccount", Plural: "serviceaccounts"},
}

func DefaultResourceTypeKeys() []string {
	keys := make([]string, 0, len(ResourceTypeOptions))
	for _, option := range ResourceTypeOptions {
		if option.SelectedByDefault {
			keys = append(keys, option.Key)
		}
	}
	return keys
}

func SelectedResourceTypes(keys []string) []ResourceTypeOption {
	selected := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		selected[key] = struct{}{}
	}
	options := make([]ResourceTypeOption, 0, len(keys))
	for _, option := range ResourceTypeOptions {
		if _, ok := selected[option.Key]; ok {
			options = append(options, option)
		}
	}
	return options
}

func ToGVK(option ResourceTypeOption) types.GroupVersionKind {
	return types.GroupVersionKind{
		Group:   option.Group,
		Version: option.Version,
		Kind:    option.Kind,
	}
}

func IsCurated(apiVersion, kind string) bool {
	_, ok := FindByAPIVersionKind(apiVersion, kind)
	return ok
}

func FindByAPIVersionKind(apiVersion, kind string) (ResourceTypeOption, bool) {
	for _, option := range ResourceTypeOptions {
		if option.Kind == kind && option.APIVersion() == apiVersion {
			return option, true
		}
	}
	return ResourceTypeOption{}, false
}

func FindByKindName(value string) (ResourceTypeOption, bool) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	for _, option := range ResourceTypeOptions {
		if strings.ToLower(option.Kind) == normalized || strings.ToLower(option.Key) == normalized || strings.ToLower(option.Label) == normalized {
			return option, true
		}
	}
	return ResourceTypeOption{}, false
}

func (r ResourceTypeOption) APIVersion() string {
	if r.Group == "" {
		return r.Version
	}
	return r.Group + "/" + r.Version
}
