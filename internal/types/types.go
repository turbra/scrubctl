package types

type Condition struct {
	Type               string `json:"type" yaml:"type"`
	Status             string `json:"status" yaml:"status"`
	Reason             string `json:"reason" yaml:"reason"`
	Message            string `json:"message" yaml:"message"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty" yaml:"lastTransitionTime,omitempty"`
}

type GroupVersionKind struct {
	Group   string `json:"group,omitempty" yaml:"group,omitempty"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	Kind    string `json:"kind" yaml:"kind"`
}

type ResourceSummary struct {
	Total               int `json:"total,omitempty" yaml:"total,omitempty"`
	Included            int `json:"included,omitempty" yaml:"included,omitempty"`
	IncludedWithCleanup int `json:"includedWithCleanup,omitempty" yaml:"includedWithCleanup,omitempty"`
	NeedsReview         int `json:"needsReview,omitempty" yaml:"needsReview,omitempty"`
	Excluded            int `json:"excluded,omitempty" yaml:"excluded,omitempty"`
}

type Classification string

const (
	ClassificationInclude Classification = "include"
	ClassificationCleanup Classification = "cleanup"
	ClassificationReview  Classification = "review"
	ClassificationExclude Classification = "exclude"
)

type ResourceObject map[string]any

type ResourceClassification struct {
	APIVersion        string         `json:"apiVersion" yaml:"apiVersion"`
	Kind              string         `json:"kind" yaml:"kind"`
	Name              string         `json:"name" yaml:"name"`
	Namespace         string         `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Classification    Classification `json:"classification" yaml:"classification"`
	Reason            string         `json:"reason" yaml:"reason"`
	Preview           string         `json:"preview,omitempty" yaml:"preview,omitempty"`
	SanitizedResource ResourceObject `json:"sanitizedResource,omitempty" yaml:"sanitizedResource,omitempty"`
}

type NamespaceScan struct {
	Metadata NamespaceScanMetadata `json:"metadata" yaml:"metadata"`
	Spec     NamespaceScanSpec     `json:"spec" yaml:"spec"`
	Status   NamespaceScanStatus   `json:"status" yaml:"status"`
}

type NamespaceScanMetadata struct {
	Namespace string `json:"namespace" yaml:"namespace"`
	ScannedAt string `json:"scannedAt" yaml:"scannedAt"`
}

type NamespaceScanSpec struct {
	Namespace            string             `json:"namespace" yaml:"namespace"`
	IncludeResourceTypes []GroupVersionKind `json:"includeResourceTypes,omitempty" yaml:"includeResourceTypes,omitempty"`
	SecretHandling       string             `json:"secretHandling,omitempty" yaml:"secretHandling,omitempty"`
}

type NamespaceScanStatus struct {
	Phase           string                   `json:"phase" yaml:"phase"`
	ResourceSummary ResourceSummary          `json:"resourceSummary" yaml:"resourceSummary"`
	ResourceDetails []ResourceClassification `json:"resourceDetails" yaml:"resourceDetails"`
	Conditions      []Condition              `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

func (r ResourceObject) APIVersion() string {
	value, _ := r["apiVersion"].(string)
	return value
}

func (r ResourceObject) Kind() string {
	value, _ := r["kind"].(string)
	return value
}

func (r ResourceObject) Metadata() map[string]any {
	value, _ := r["metadata"].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func (r ResourceObject) Name() string {
	value, _ := r.Metadata()["name"].(string)
	return value
}

func (r ResourceObject) Namespace() string {
	value, _ := r.Metadata()["namespace"].(string)
	return value
}
