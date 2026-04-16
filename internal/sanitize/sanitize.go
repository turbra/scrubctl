package sanitize

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/turbra/scrubctl/internal/types"
)

const MaxPreviewBytes = 16 * 1024

func BuildPreview(resource types.ResourceObject, classification types.Classification, secretHandling string) string {
	return BuildPreviewFromSanitized(SanitizeResource(resource, classification, secretHandling))
}

func SanitizeResource(resource types.ResourceObject, classification types.Classification, secretHandling string) types.ResourceObject {
	if classification == types.ClassificationExclude {
		return nil
	}
	return sanitizeForExport(resource, secretHandling)
}

func BuildPreviewFromSanitized(sanitized types.ResourceObject) string {
	if sanitized == nil {
		return ""
	}
	yamlText := SerializeResource(sanitized)
	if len(yamlText) <= MaxPreviewBytes {
		return yamlText
	}
	truncated := yamlText[:MaxPreviewBytes]
	lastNewline := -1
	for i := len(truncated) - 1; i >= 0; i-- {
		if truncated[i] == '\n' {
			lastNewline = i
			break
		}
	}
	if lastNewline > 0 {
		truncated = truncated[:lastNewline]
	}
	return truncated + "\n# Preview truncated\n"
}

func SerializeResource(resource types.ResourceObject) string {
	data, err := yaml.Marshal(resource)
	if err != nil {
		panic("sanitize: failed to marshal resource to YAML: " + err.Error())
	}
	return string(data)
}

func sanitizeForExport(resource types.ResourceObject, secretHandling string) types.ResourceObject {
	if resource.Kind() == "Secret" && secretHandling == "omit" {
		return nil
	}

	sanitized := deepCopy(resource)
	sanitizeMetadata(sanitized)
	sanitizeTopLevelDefaults(sanitized)
	sanitizeKindSpecificFields(sanitized, secretHandling)
	return sanitized
}

func deepCopy(resource types.ResourceObject) types.ResourceObject {
	data, err := json.Marshal(resource)
	if err != nil {
		panic("sanitize: failed to marshal resource for deep copy: " + err.Error())
	}
	var copied types.ResourceObject
	if err := json.Unmarshal(data, &copied); err != nil {
		panic("sanitize: failed to unmarshal resource for deep copy: " + err.Error())
	}
	return copied
}

func sanitizeMetadata(resource types.ResourceObject) {
	unstructured.RemoveNestedField(resource, "metadata", "uid")
	unstructured.RemoveNestedField(resource, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(resource, "metadata", "generation")
	unstructured.RemoveNestedField(resource, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(resource, "metadata", "managedFields")
	unstructured.RemoveNestedField(resource, "metadata", "selfLink")
	unstructured.RemoveNestedField(resource, "metadata", "ownerReferences")

	if annotations, found, _ := unstructured.NestedStringMap(resource, "metadata", "annotations"); found {
		for key := range annotations {
			if shouldStripAnnotation(key) {
				delete(annotations, key)
			}
		}
		if len(annotations) == 0 {
			unstructured.RemoveNestedField(resource, "metadata", "annotations")
		} else {
			_ = unstructured.SetNestedStringMap(resource, annotations, "metadata", "annotations")
		}
	}

	if finalizers, found, _ := unstructured.NestedStringSlice(resource, "metadata", "finalizers"); found {
		filtered := make([]string, 0, len(finalizers))
		for _, finalizer := range finalizers {
			if !isSystemFinalizer(finalizer) {
				filtered = append(filtered, finalizer)
			}
		}
		if len(filtered) == 0 {
			unstructured.RemoveNestedField(resource, "metadata", "finalizers")
		} else {
			_ = unstructured.SetNestedStringSlice(resource, filtered, "metadata", "finalizers")
		}
	}
}

func sanitizeTopLevelDefaults(resource types.ResourceObject) {
	delete(resource, "status")

	if spec, ok := resource["spec"].(map[string]any); ok {
		delete(spec, "nodeName")
		if spec["schedulerName"] == "default-scheduler" {
			delete(spec, "schedulerName")
		}
	}

	sanitizePodTemplate(resource)
}

func sanitizeKindSpecificFields(resource types.ResourceObject, secretHandling string) {
	switch resource.Kind() {
	case "Deployment":
		sanitizeDeploymentDefaults(resource)
	case "StatefulSet":
		sanitizeStatefulSetDefaults(resource)
	case "DaemonSet":
		sanitizeDaemonSetDefaults(resource)
	case "Job":
		sanitizeJobDefaults(resource)
	case "CronJob":
		sanitizeCronJobDefaults(resource)
	case "Secret":
		sanitizeSecret(resource, secretHandling)
	case "PersistentVolumeClaim":
		unstructured.RemoveNestedField(resource, "spec", "volumeName")
	case "Service":
		unstructured.RemoveNestedField(resource, "spec", "clusterIP")
		unstructured.RemoveNestedField(resource, "spec", "clusterIPs")
		unstructured.RemoveNestedField(resource, "spec", "ipFamilies")
		unstructured.RemoveNestedField(resource, "spec", "ipFamilyPolicy")
		unstructured.RemoveNestedField(resource, "spec", "internalTrafficPolicy")
		if sessionAffinity, found, _ := unstructured.NestedString(resource, "spec", "sessionAffinity"); found && sessionAffinity == "None" {
			unstructured.RemoveNestedField(resource, "spec", "sessionAffinity")
		}
		if serviceType, found, _ := unstructured.NestedString(resource, "spec", "type"); found && serviceType == "LoadBalancer" {
			sanitizeServiceAnnotations(resource)
		}
	case "ServiceAccount":
		sanitizeServiceAccountDefaults(resource)
	case "Route":
		sanitizeRouteDefaults(resource)
	case "BuildConfig":
		sanitizeBuildConfigDefaults(resource)
	case "ImageStream":
		unstructured.RemoveNestedField(resource, "spec", "dockerImageRepository")
	}
}

func sanitizeDeploymentDefaults(resource types.ResourceObject) {
	if numberEquals(resource, 600, "spec", "progressDeadlineSeconds") {
		unstructured.RemoveNestedField(resource, "spec", "progressDeadlineSeconds")
	}
	if numberEquals(resource, 10, "spec", "revisionHistoryLimit") {
		unstructured.RemoveNestedField(resource, "spec", "revisionHistoryLimit")
	}
	if numberEquals(resource, 0, "spec", "minReadySeconds") {
		unstructured.RemoveNestedField(resource, "spec", "minReadySeconds")
	}

	strategy, found, _ := unstructured.NestedMap(resource, "spec", "strategy")
	if !found {
		return
	}
	rollingUpdate, foundRolling := strategy["rollingUpdate"].(map[string]any)
	hasDefaultType := strategy["type"] == "RollingUpdate"
	hasDefaultRolling := foundRolling && rollingUpdate["maxSurge"] == "25%" && rollingUpdate["maxUnavailable"] == "25%"
	if hasDefaultType && hasDefaultRolling {
		unstructured.RemoveNestedField(resource, "spec", "strategy")
	}
}

func sanitizeStatefulSetDefaults(resource types.ResourceObject) {
	if numberEquals(resource, 10, "spec", "revisionHistoryLimit") {
		unstructured.RemoveNestedField(resource, "spec", "revisionHistoryLimit")
	}
	if numberEquals(resource, 0, "spec", "minReadySeconds") {
		unstructured.RemoveNestedField(resource, "spec", "minReadySeconds")
	}
	if podMgmt, found, _ := unstructured.NestedString(resource, "spec", "podManagementPolicy"); found && podMgmt == "OrderedReady" {
		unstructured.RemoveNestedField(resource, "spec", "podManagementPolicy")
	}

	updateStrategy, found, _ := unstructured.NestedMap(resource, "spec", "updateStrategy")
	if found {
		rollingUpdate, foundRolling := updateStrategy["rollingUpdate"].(map[string]any)
		hasDefaultType := updateStrategy["type"] == "RollingUpdate"
		hasDefaultRolling := foundRolling && numberEqualsAny(rollingUpdate["partition"], 0)
		if hasDefaultType && hasDefaultRolling {
			unstructured.RemoveNestedField(resource, "spec", "updateStrategy")
		}
	}

	sanitizeVolumeClaimTemplates(resource)
}

func sanitizeVolumeClaimTemplates(resource types.ResourceObject) {
	vcts, found, _ := unstructured.NestedSlice(resource, "spec", "volumeClaimTemplates")
	if !found {
		return
	}
	changed := false
	for _, vctAny := range vcts {
		vct, ok := vctAny.(map[string]any)
		if !ok {
			continue
		}
		if metadata, ok := vct["metadata"].(map[string]any); ok {
			if _, has := metadata["creationTimestamp"]; has {
				delete(metadata, "creationTimestamp")
				changed = true
				if len(metadata) == 0 {
					delete(vct, "metadata")
				}
			}
		}
	}
	if changed {
		_ = unstructured.SetNestedSlice(resource, vcts, "spec", "volumeClaimTemplates")
	}
}

func sanitizeDaemonSetDefaults(resource types.ResourceObject) {
	if numberEquals(resource, 10, "spec", "revisionHistoryLimit") {
		unstructured.RemoveNestedField(resource, "spec", "revisionHistoryLimit")
	}
	if numberEquals(resource, 0, "spec", "minReadySeconds") {
		unstructured.RemoveNestedField(resource, "spec", "minReadySeconds")
	}

	updateStrategy, found, _ := unstructured.NestedMap(resource, "spec", "updateStrategy")
	if !found {
		return
	}
	rollingUpdate, foundRolling := updateStrategy["rollingUpdate"].(map[string]any)
	hasDefaultType := updateStrategy["type"] == "RollingUpdate"
	hasDefaultRolling := foundRolling && numberEqualsAny(rollingUpdate["maxUnavailable"], 1) && numberEqualsAny(rollingUpdate["maxSurge"], 0)
	if hasDefaultType && hasDefaultRolling {
		unstructured.RemoveNestedField(resource, "spec", "updateStrategy")
	}
}

func sanitizeJobDefaults(resource types.ResourceObject) {
	if numberEquals(resource, 6, "spec", "backoffLimit") {
		unstructured.RemoveNestedField(resource, "spec", "backoffLimit")
	}
	if numberEquals(resource, 1, "spec", "completions") {
		unstructured.RemoveNestedField(resource, "spec", "completions")
	}
	if numberEquals(resource, 1, "spec", "parallelism") {
		unstructured.RemoveNestedField(resource, "spec", "parallelism")
	}
	if completionMode, found, _ := unstructured.NestedString(resource, "spec", "completionMode"); found && completionMode == "NonIndexed" {
		unstructured.RemoveNestedField(resource, "spec", "completionMode")
	}
	if suspend, found, _ := unstructured.NestedBool(resource, "spec", "suspend"); found && !suspend {
		unstructured.RemoveNestedField(resource, "spec", "suspend")
	}
}

func sanitizeCronJobDefaults(resource types.ResourceObject) {
	if concurrency, found, _ := unstructured.NestedString(resource, "spec", "concurrencyPolicy"); found && concurrency == "Allow" {
		unstructured.RemoveNestedField(resource, "spec", "concurrencyPolicy")
	}
	if suspend, found, _ := unstructured.NestedBool(resource, "spec", "suspend"); found && !suspend {
		unstructured.RemoveNestedField(resource, "spec", "suspend")
	}
	if numberEquals(resource, 3, "spec", "successfulJobsHistoryLimit") {
		unstructured.RemoveNestedField(resource, "spec", "successfulJobsHistoryLimit")
	}
	if numberEquals(resource, 1, "spec", "failedJobsHistoryLimit") {
		unstructured.RemoveNestedField(resource, "spec", "failedJobsHistoryLimit")
	}
	cleanNestedMetadata(resource, "spec", "jobTemplate", "metadata")
	sanitizeNestedJobDefaults(resource, "spec", "jobTemplate", "spec")
}

func sanitizeNestedJobDefaults(resource types.ResourceObject, fields ...string) {
	jobSpec, found, _ := unstructured.NestedMap(resource, fields...)
	if !found {
		return
	}
	if numberEqualsAny(jobSpec["backoffLimit"], 6) {
		delete(jobSpec, "backoffLimit")
	}
	if numberEqualsAny(jobSpec["completions"], 1) {
		delete(jobSpec, "completions")
	}
	if numberEqualsAny(jobSpec["parallelism"], 1) {
		delete(jobSpec, "parallelism")
	}
	if jobSpec["completionMode"] == "NonIndexed" {
		delete(jobSpec, "completionMode")
	}
	if suspend, ok := jobSpec["suspend"].(bool); ok && !suspend {
		delete(jobSpec, "suspend")
	}
	_ = unstructured.SetNestedMap(resource, jobSpec, fields...)
}

func sanitizeServiceAccountDefaults(resource types.ResourceObject) {
	unstructured.RemoveNestedField(resource, "secrets")
}

func sanitizeRouteDefaults(resource types.ResourceObject) {
	if wildcardPolicy, found, _ := unstructured.NestedString(resource, "spec", "wildcardPolicy"); found && wildcardPolicy == "None" {
		unstructured.RemoveNestedField(resource, "spec", "wildcardPolicy")
	}
}

func sanitizeBuildConfigDefaults(resource types.ResourceObject) {
	if runPolicy, found, _ := unstructured.NestedString(resource, "spec", "runPolicy"); found && runPolicy == "Serial" {
		unstructured.RemoveNestedField(resource, "spec", "runPolicy")
	}
}

func sanitizeSecret(resource types.ResourceObject, secretHandling string) {
	if secretHandling == "include" {
		return
	}
	redactStringMap(resource, "data")
	redactStringMap(resource, "stringData")
}

func redactStringMap(resource types.ResourceObject, field string) {
	value, ok := resource[field].(map[string]any)
	if !ok {
		return
	}
	for key := range value {
		value[key] = "<REDACTED>"
	}
}

func sanitizeServiceAnnotations(resource types.ResourceObject) {
	annotations, found, _ := unstructured.NestedStringMap(resource, "metadata", "annotations")
	if !found {
		return
	}
	for key := range annotations {
		if hasAnyPrefix(key, "service.beta.kubernetes.io/", "service.kubernetes.io/", "metallb.universe.tf/") {
			delete(annotations, key)
		}
	}
	if len(annotations) == 0 {
		unstructured.RemoveNestedField(resource, "metadata", "annotations")
	} else {
		_ = unstructured.SetNestedStringMap(resource, annotations, "metadata", "annotations")
	}
}

func shouldStripAnnotation(key string) bool {
	switch key {
	case "kubectl.kubernetes.io/last-applied-configuration", "deployment.kubernetes.io/revision",
		"openshift.io/generated-by", "openshift.io/host.generated", "openshift.io/required-scc":
		return true
	default:
		return hasAnyPrefix(key, "pv.kubernetes.io/", "operator.openshift.io/", "openshift.io/build.", "imageregistry.operator.openshift.io/", "volume.beta.kubernetes.io/", "volume.kubernetes.io/")
	}
}

func sanitizePodTemplate(resource types.ResourceObject) {
	cleanPodTemplateAt(resource, "spec", "template")
	cleanPodTemplateAt(resource, "spec", "jobTemplate", "spec", "template")
}

func cleanPodTemplateAt(resource types.ResourceObject, fields ...string) {
	template, found, _ := unstructured.NestedMap(resource, fields...)
	if !found {
		return
	}
	if metadata, ok := template["metadata"].(map[string]any); ok {
		delete(metadata, "creationTimestamp")
		if len(metadata) == 0 {
			delete(template, "metadata")
		}
	}

	spec, ok := template["spec"].(map[string]any)
	if !ok {
		_ = unstructured.SetNestedMap(resource, template, fields...)
		return
	}
	if spec["schedulerName"] == "default-scheduler" {
		delete(spec, "schedulerName")
	}
	if spec["dnsPolicy"] == "ClusterFirst" {
		delete(spec, "dnsPolicy")
	}
	if spec["restartPolicy"] == "Always" {
		delete(spec, "restartPolicy")
	}
	if securityContext, ok := spec["securityContext"].(map[string]any); ok && len(securityContext) == 0 {
		delete(spec, "securityContext")
	}
	if affinity, ok := spec["affinity"].(map[string]any); ok && len(affinity) == 0 {
		delete(spec, "affinity")
	}
	if containers, ok := spec["containers"].([]any); ok {
		for _, containerAny := range containers {
			container, ok := containerAny.(map[string]any)
			if !ok {
				continue
			}
			if container["terminationMessagePath"] == "/dev/termination-log" {
				delete(container, "terminationMessagePath")
			}
			if container["terminationMessagePolicy"] == "File" {
				delete(container, "terminationMessagePolicy")
			}
		}
	}
	_ = unstructured.SetNestedMap(resource, template, fields...)
}

func cleanNestedMetadata(resource types.ResourceObject, fields ...string) {
	metadata, found, _ := unstructured.NestedMap(resource, fields...)
	if !found {
		return
	}
	delete(metadata, "creationTimestamp")
	if len(metadata) == 0 {
		unstructured.RemoveNestedField(resource, fields...)
	} else {
		_ = unstructured.SetNestedMap(resource, metadata, fields...)
	}
}

func isSystemFinalizer(finalizer string) bool {
	return hasAnyPrefix(finalizer, "kubernetes.io/", "openshift.io/", "operator.openshift.io/")
}

func hasAnyPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func numberEqualsAny(value any, expected float64) bool {
	switch typed := value.(type) {
	case int:
		return float64(typed) == expected
	case int32:
		return float64(typed) == expected
	case int64:
		return float64(typed) == expected
	case float32:
		return float64(typed) == expected
	case float64:
		return typed == expected
	default:
		return false
	}
}

func numberEquals(resource types.ResourceObject, expected float64, fields ...string) bool {
	value, found, err := unstructured.NestedFieldNoCopy(resource, fields...)
	if err != nil || !found {
		return false
	}
	switch typed := value.(type) {
	case int:
		return float64(typed) == expected
	case int32:
		return float64(typed) == expected
	case int64:
		return float64(typed) == expected
	case float32:
		return float64(typed) == expected
	case float64:
		return typed == expected
	default:
		return false
	}
}
