package scan

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/turbra/scrubctl/internal/classify"
	"github.com/turbra/scrubctl/internal/resources"
	"github.com/turbra/scrubctl/internal/sanitize"
	"github.com/turbra/scrubctl/internal/types"
)

var ignoredListErrorCodes = map[int]struct{}{
	http.StatusUnauthorized:     {},
	http.StatusForbidden:        {},
	http.StatusNotFound:         {},
	http.StatusMethodNotAllowed: {},
}

type Options struct {
	Kubeconfig     string
	Context        string
	Namespace      string
	SecretHandling string
	IncludeTypes   []resources.ResourceTypeOption
	Now            func() time.Time
}

func Run(ctx context.Context, options Options) (types.NamespaceScan, error) {
	if options.Namespace == "" {
		return types.NamespaceScan{}, errors.New("namespace is required")
	}

	client, err := newDynamicClient(options.Kubeconfig, options.Context)
	if err != nil {
		return types.NamespaceScan{}, err
	}

	resourceDetails := make([]types.ResourceClassification, 0)
	for _, resourceType := range options.IncludeTypes {
		items, err := listNamespaced(ctx, client, resourceType, options.Namespace)
		if err != nil {
			if shouldIgnoreListError(err) {
				continue
			}
			return types.NamespaceScan{}, err
		}

		for _, item := range items {
			resource := types.ResourceObject(item.Object)
			classificationResult := classify.ClassifyCuratedResource(resource)
			sanitizedResource := sanitize.SanitizeResource(resource, classificationResult.Classification, options.SecretHandling)
			classificationResult.SanitizedResource = sanitizedResource
			classificationResult.Preview = sanitize.BuildPreviewFromSanitized(sanitizedResource)
			resourceDetails = append(resourceDetails, classificationResult)
		}
	}

	resourceDetails = classify.SortResourceDetails(resourceDetails)
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}

	return types.NamespaceScan{
		Metadata: types.NamespaceScanMetadata{
			Namespace: options.Namespace,
			ScannedAt: now().UTC().Format(time.RFC3339),
		},
		Spec: types.NamespaceScanSpec{
			Namespace:            options.Namespace,
			SecretHandling:       options.SecretHandling,
			IncludeResourceTypes: toGVKs(options.IncludeTypes),
		},
		Status: types.NamespaceScanStatus{
			Phase:           "Completed",
			ResourceSummary: classify.SummarizeResourceDetails(resourceDetails),
			ResourceDetails: resourceDetails,
			Conditions: []types.Condition{{
				Type:    "Completed",
				Status:  "True",
				Reason:  "LocalNamespaceScanCompleted",
				Message: fmt.Sprintf("Scanned %d selected resource kinds in %s.", len(options.IncludeTypes), options.Namespace),
			}},
		},
	}, nil
}

func newClientConfig(kubeconfigPath, kubeContext string) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}
	overrides := &clientcmd.ConfigOverrides{}
	if kubeContext != "" {
		overrides.CurrentContext = kubeContext
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
}

func newDynamicClient(kubeconfigPath, kubeContext string) (dynamic.Interface, error) {
	config, err := newClientConfig(kubeconfigPath, kubeContext).ClientConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}

func ResolveNamespace(kubeconfigPath, kubeContext, namespaceOverride string) (string, error) {
	if namespaceOverride != "" {
		return namespaceOverride, nil
	}

	namespace, _, err := newClientConfig(kubeconfigPath, kubeContext).Namespace()
	if err != nil {
		return "", err
	}
	if namespace == "" {
		return "", errors.New("namespace is required")
	}
	return namespace, nil
}

func listNamespaced(ctx context.Context, client dynamic.Interface, option resources.ResourceTypeOption, namespace string) ([]unstructured.Unstructured, error) {
	gvr := schema.GroupVersionResource{
		Group:    option.Group,
		Version:  option.Version,
		Resource: option.Plural,
	}
	list, err := client.Resource(gvr).Namespace(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func shouldIgnoreListError(err error) bool {
	status := 0
	if apiStatus, ok := err.(apierrors.APIStatus); ok {
		status = int(apiStatus.Status().Code)
	}
	if status == 0 {
		var statusErr *apierrors.StatusError
		if errors.As(err, &statusErr) {
			status = int(statusErr.ErrStatus.Code)
		}
	}
	_, ok := ignoredListErrorCodes[status]
	return ok
}

func toGVKs(options []resources.ResourceTypeOption) []types.GroupVersionKind {
	values := make([]types.GroupVersionKind, 0, len(options))
	for _, option := range options {
		values = append(values, resources.ToGVK(option))
	}
	return values
}
