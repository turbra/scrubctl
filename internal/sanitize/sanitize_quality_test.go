package sanitize_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/turbra/scrubctl/internal/classify"
	"github.com/turbra/scrubctl/internal/sanitize"
	"github.com/turbra/scrubctl/internal/types"
)

var forbiddenTopMeta = []string{
	"uid", "resourceVersion", "generation", "creationTimestamp",
	"managedFields", "selfLink", "ownerReferences",
}

var forbiddenAnnotationPrefixes = []string{
	"kubectl.kubernetes.io/last-applied-configuration",
	"pv.kubernetes.io/",
	"operator.openshift.io/",
	"openshift.io/build.",
	"imageregistry.operator.openshift.io/",
	"volume.beta.kubernetes.io/",
	"volume.kubernetes.io/",
}

func TestSanitizationQuality(t *testing.T) {
	fixtureDirs, err := filepath.Glob(filepath.Join("..", "..", "testdata", "fixtures", "*"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(fixtureDirs)

	for _, fixtureDir := range fixtureDirs {
		info, err := os.Stat(fixtureDir)
		if err != nil || !info.IsDir() {
			continue
		}
		name := filepath.Base(fixtureDir)
		t.Run(name, func(t *testing.T) {
			inputPath := filepath.Join(fixtureDir, "input.yaml")
			if _, err := os.Stat(inputPath); err != nil {
				t.Skip("no input.yaml")
			}

			resource := readResource(t, inputPath)
			result := classify.ClassifyCuratedResource(resource)
			if result.Classification == types.ClassificationExclude {
				t.Skip("excluded resource")
			}

			sanitized := sanitize.SanitizeResource(resource, result.Classification, "redact")
			if sanitized == nil {
				t.Skip("sanitized to nil")
			}

			checkForbiddenMetadata(t, sanitized)
			checkForbiddenAnnotations(t, sanitized)
			checkNoStatus(t, sanitized)
			walkForbiddenPatterns(t, sanitized, "")
		})
	}
}

func checkForbiddenMetadata(t *testing.T, resource types.ResourceObject) {
	t.Helper()
	metadata, ok := resource["metadata"].(map[string]any)
	if !ok {
		return
	}
	for _, field := range forbiddenTopMeta {
		if _, exists := metadata[field]; exists {
			t.Errorf("metadata.%s should be stripped", field)
		}
	}
}

func checkForbiddenAnnotations(t *testing.T, resource types.ResourceObject) {
	t.Helper()
	metadata, ok := resource["metadata"].(map[string]any)
	if !ok {
		return
	}
	annotations, ok := metadata["annotations"].(map[string]any)
	if !ok {
		return
	}
	for key := range annotations {
		for _, prefix := range forbiddenAnnotationPrefixes {
			if strings.Contains(prefix, "/") && !strings.HasSuffix(prefix, "/") {
				if key == prefix {
					t.Errorf("annotation %q should be stripped", key)
				}
			} else if strings.HasPrefix(key, prefix) {
				t.Errorf("annotation %q should be stripped (matches prefix %q)", key, prefix)
			}
		}
	}
}

func checkNoStatus(t *testing.T, resource types.ResourceObject) {
	t.Helper()
	if _, exists := resource["status"]; exists {
		t.Error("status should be stripped")
	}
}

func walkForbiddenPatterns(t *testing.T, value any, path string) {
	t.Helper()
	switch v := value.(type) {
	case map[string]any:
		if path != "" {
			if len(v) == 0 {
				key := lastPathComponent(path)
				if key == "securityContext" || key == "affinity" {
					t.Errorf("%s: empty %s map should be stripped", path, key)
				}
			}
		}
		for key, child := range v {
			childPath := joinPath(path, key)
			if key == "creationTimestamp" && child == nil {
				t.Errorf("%s: null creationTimestamp should be stripped", childPath)
			}
			walkForbiddenPatterns(t, child, childPath)
		}
	case []any:
		for i, child := range v {
			walkForbiddenPatterns(t, child, fmt.Sprintf("%s[%d]", path, i))
		}
	}
}

func lastPathComponent(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i+1:]
	}
	return path
}

func joinPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func readResource(t *testing.T, path string) types.ResourceObject {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var resource types.ResourceObject
	if err := yaml.Unmarshal(data, &resource); err != nil {
		t.Fatal(err)
	}
	return resource
}
