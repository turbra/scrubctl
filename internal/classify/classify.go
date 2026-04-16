package classify

import (
	"sort"
	"strings"

	"github.com/turbra/scrubctl/internal/openshift"
	"github.com/turbra/scrubctl/internal/resources"
	"github.com/turbra/scrubctl/internal/types"
)

func ClassifyCuratedResource(resource types.ResourceObject) types.ResourceClassification {
	if !resources.IsCurated(resource.APIVersion(), resource.Kind()) {
		return newClassification(resource, types.ClassificationExclude, "kind not in curated resource set")
	}
	return ClassifyResource(resource)
}

func ClassifyForDirectScrub(resource types.ResourceObject) types.ResourceClassification {
	return classifyForScrub(resource)
}

func classifyForScrub(resource types.ResourceObject) types.ResourceClassification {
	if resource.APIVersion() == "gitops.stakkr.io/v1alpha1" {
		return newClassification(resource, types.ClassificationExclude, "GitOps Exporter control-plane resource")
	}

	if metadata := resource.Metadata(); metadata != nil {
		if ownerRefs, ok := metadata["ownerReferences"].([]any); ok && len(ownerRefs) > 0 {
			return newClassification(resource, types.ClassificationReview, "Controller-owned resource")
		}
		if labels, ok := metadata["labels"].(map[string]any); ok {
			if managedBy, ok := labels["app.kubernetes.io/managed-by"].(string); ok && strings.ToLower(managedBy) == "helm" {
				return newClassification(resource, types.ClassificationReview, "Helm-managed lifecycle detected")
			}
		}
	}

	if openshift.IsScaffoldingResource(resource) {
		return newClassification(resource, types.ClassificationReview, "OpenShift-injected namespace scaffolding")
	}

	switch resource.Kind() {
	case "Pod", "ReplicaSet", "EndpointSlice", "Endpoints", "Event", "ControllerRevision",
		"Lease", "TokenReview", "SubjectAccessReview", "Node", "PersistentVolume":
		return newClassification(resource, types.ClassificationReview, "Runtime or cluster-owned resource; review before use")
	case "Secret", "PodDisruptionBudget", "ResourceQuota", "LimitRange", "DeploymentConfig":
		return newClassification(resource, types.ClassificationReview, reviewReasonForKind(resource.Kind()))
	case "PersistentVolumeClaim", "ImageStream", "ImageStreamTag":
		return newClassification(resource, types.ClassificationCleanup, cleanupReasonForKind(resource.Kind()))
	case "Service":
		if spec, ok := resource["spec"].(map[string]any); ok {
			if serviceType, ok := spec["type"].(string); ok && serviceType == "LoadBalancer" {
				return newClassification(resource, types.ClassificationCleanup, "LoadBalancer service needs environment-specific cleanup")
			}
		}
		return newClassification(resource, types.ClassificationInclude, "Declarative service resource")
	default:
		return newClassification(resource, types.ClassificationInclude, includeReasonForKind(resource.Kind()))
	}
}

func ClassifyResource(resource types.ResourceObject) types.ResourceClassification {
	if resource.APIVersion() == "gitops.stakkr.io/v1alpha1" {
		return newClassification(resource, types.ClassificationExclude, "GitOps Exporter control-plane resource")
	}

	if metadata := resource.Metadata(); metadata != nil {
		if ownerRefs, ok := metadata["ownerReferences"].([]any); ok && len(ownerRefs) > 0 {
			return newClassification(resource, types.ClassificationExclude, "Controller-owned resource")
		}

		if labels, ok := metadata["labels"].(map[string]any); ok {
			if managedBy, ok := labels["app.kubernetes.io/managed-by"].(string); ok && strings.ToLower(managedBy) == "helm" {
				return newClassification(resource, types.ClassificationReview, "Helm-managed lifecycle detected")
			}
		}
	}

	if openshift.IsScaffoldingResource(resource) {
		return newClassification(resource, types.ClassificationExclude, "OpenShift-injected namespace scaffolding")
	}

	switch resource.Kind() {
	case "Pod", "ReplicaSet", "EndpointSlice", "Endpoints", "Event", "ControllerRevision", "Lease", "TokenReview", "SubjectAccessReview", "Node", "PersistentVolume":
		return newClassification(resource, types.ClassificationExclude, "Runtime-generated or cluster-owned resource")
	case "Secret", "PodDisruptionBudget", "ResourceQuota", "LimitRange", "DeploymentConfig":
		return newClassification(resource, types.ClassificationReview, reviewReasonForKind(resource.Kind()))
	case "PersistentVolumeClaim", "ImageStream", "ImageStreamTag":
		return newClassification(resource, types.ClassificationCleanup, cleanupReasonForKind(resource.Kind()))
	case "Service":
		if spec, ok := resource["spec"].(map[string]any); ok {
			if serviceType, ok := spec["type"].(string); ok && serviceType == "LoadBalancer" {
				return newClassification(resource, types.ClassificationCleanup, "LoadBalancer service needs environment-specific cleanup")
			}
		}
		return newClassification(resource, types.ClassificationInclude, "Declarative service resource")
	case "Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob", "ConfigMap", "ServiceAccount", "Role", "RoleBinding", "NetworkPolicy", "HorizontalPodAutoscaler", "Route", "BuildConfig", "Ingress":
		return newClassification(resource, types.ClassificationInclude, includeReasonForKind(resource.Kind()))
	default:
		return newClassification(resource, types.ClassificationReview, "No explicit classifier yet; review before export")
	}
}

func SortResourceDetails(resourcesIn []types.ResourceClassification) []types.ResourceClassification {
	resourcesOut := append([]types.ResourceClassification(nil), resourcesIn...)
	sort.Slice(resourcesOut, func(i, j int) bool {
		left := resourcesOut[i]
		right := resourcesOut[j]
		if classificationRank(left.Classification) != classificationRank(right.Classification) {
			return classificationRank(left.Classification) < classificationRank(right.Classification)
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.Name < right.Name
	})
	return resourcesOut
}

func SummarizeResourceDetails(resourcesIn []types.ResourceClassification) types.ResourceSummary {
	summary := types.ResourceSummary{}
	for _, resource := range resourcesIn {
		summary.Total++
		switch resource.Classification {
		case types.ClassificationInclude:
			summary.Included++
		case types.ClassificationCleanup:
			summary.IncludedWithCleanup++
		case types.ClassificationReview:
			summary.NeedsReview++
		default:
			summary.Excluded++
		}
	}
	return summary
}

func newClassification(resource types.ResourceObject, classification types.Classification, reason string) types.ResourceClassification {
	return types.ResourceClassification{
		APIVersion:     resource.APIVersion(),
		Kind:           resource.Kind(),
		Name:           resource.Name(),
		Namespace:      resource.Namespace(),
		Classification: classification,
		Reason:         reason,
	}
}

func includeReasonForKind(kind string) string {
	switch kind {
	case "Route":
		return "OpenShift route can be declared directly"
	case "BuildConfig":
		return "OpenShift build configuration can be declared directly"
	case "ConfigMap":
		return "Configuration resource is declarative"
	case "ServiceAccount", "Role", "RoleBinding", "NetworkPolicy":
		return "Namespaced access or policy resource is declarative"
	case "HorizontalPodAutoscaler":
		return "Scaling policy resource is declarative"
	case "Ingress":
		return "Ingress routing resource is declarative"
	default:
		return "Workload resource is declarative"
	}
}

func cleanupReasonForKind(kind string) string {
	switch kind {
	case "PersistentVolumeClaim":
		return "Persistent volume binding details need cleanup"
	case "ImageStream", "ImageStreamTag":
		return "OpenShift image references need cleanup before export"
	default:
		return "Resource needs cleanup before export"
	}
}

func reviewReasonForKind(kind string) string {
	switch kind {
	case "Secret":
		return "Secret values require review or redaction"
	case "PodDisruptionBudget":
		return "Availability policy should be reviewed in context"
	case "ResourceQuota", "LimitRange":
		return "Namespace policy resource may not belong in app manifests"
	case "DeploymentConfig":
		return "Legacy OpenShift deployment API should be reviewed before export"
	default:
		return "Resource should be reviewed before export"
	}
}

func classificationRank(classification types.Classification) int {
	switch classification {
	case types.ClassificationInclude:
		return 0
	case types.ClassificationCleanup:
		return 1
	case types.ClassificationReview:
		return 2
	default:
		return 3
	}
}
