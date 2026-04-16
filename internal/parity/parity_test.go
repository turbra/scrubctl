package parity_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"

	archivepkg "github.com/turbra/scrubctl/internal/archive"
	"github.com/turbra/scrubctl/internal/classify"
	"github.com/turbra/scrubctl/internal/sanitize"
	"github.com/turbra/scrubctl/internal/types"
)

type fixtureConfig struct {
	SecretHandling string `json:"secretHandling"`
	Namespace      string `json:"namespace"`
	ScannedAt      string `json:"scannedAt"`
}

func TestFixturesMatchTSExpectations(t *testing.T) {
	fixtureDirs, err := filepath.Glob(filepath.Join("..", "..", "testdata", "fixtures", "*"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(fixtureDirs)

	for _, fixtureDir := range fixtureDirs {
		if !isDirectory(fixtureDir) {
			continue
		}
		name := filepath.Base(fixtureDir)
		t.Run(name, func(t *testing.T) {
			resource := readResourceYAML(t, filepath.Join(fixtureDir, "input.yaml"))
			config := readFixtureConfig(t, fixtureDir)

			classificationResult := classify.ClassifyCuratedResource(resource)
			expectedClassification := readClassificationJSON(t, filepath.Join(fixtureDir, "expected-classification.json"))
			if diff := cmp.Diff(expectedClassification, classificationResult); diff != "" {
				t.Fatalf("classification mismatch (-want +got):\n%s", diff)
			}

			secretHandling := config.SecretHandling
			if secretHandling == "" {
				secretHandling = "redact"
			}
			sanitizedResource := sanitize.SanitizeResource(resource, classificationResult.Classification, secretHandling)
			expectedSanitized := readNormalizedYAML(t, filepath.Join(fixtureDir, "expected-sanitized.yaml"))
			if diff := cmp.Diff(expectedSanitized, normalizeValue(sanitizedResource)); diff != "" {
				t.Fatalf("sanitized resource mismatch (-want +got):\n%s", diff)
			}

			scan := buildFixtureScan(resource, config, classificationResult, sanitizedResource)
			expectedArchive := readJSONAny(t, filepath.Join(fixtureDir, "expected-archive.json"))
			actualArchive := buildArchiveExpectation(t, scan)
			if diff := cmp.Diff(expectedArchive, actualArchive); diff != "" {
				t.Fatalf("archive mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func buildFixtureScan(resource types.ResourceObject, config fixtureConfig, classificationResult types.ResourceClassification, sanitizedResource types.ResourceObject) types.NamespaceScan {
	namespace := config.Namespace
	if namespace == "" {
		namespace = resource.Namespace()
	}
	if namespace == "" {
		namespace = "default"
	}
	scannedAt := config.ScannedAt
	if scannedAt == "" {
		scannedAt = "2026-04-15T12:00:00Z"
	}
	classificationResult.Preview = sanitize.BuildPreviewFromSanitized(sanitizedResource)
	classificationResult.SanitizedResource = sanitizedResource

	includeResourceTypes := []types.GroupVersionKind{}
	if classificationResult.Classification != types.ClassificationExclude || classificationResult.Reason != "kind not in curated resource set" {
		includeResourceTypes = append(includeResourceTypes, types.GroupVersionKind{
			Group:   apiGroup(resource.APIVersion()),
			Version: apiVersion(resource.APIVersion()),
			Kind:    resource.Kind(),
		})
	}

	return types.NamespaceScan{
		Metadata: types.NamespaceScanMetadata{
			Namespace: namespace,
			ScannedAt: scannedAt,
		},
		Spec: types.NamespaceScanSpec{
			Namespace:            namespace,
			SecretHandling:       defaultString(config.SecretHandling, "redact"),
			IncludeResourceTypes: includeResourceTypes,
		},
		Status: types.NamespaceScanStatus{
			Phase:           "Completed",
			ResourceSummary: classify.SummarizeResourceDetails([]types.ResourceClassification{classificationResult}),
			ResourceDetails: []types.ResourceClassification{classificationResult},
			Conditions: []types.Condition{{
				Type:    "Completed",
				Status:  "True",
				Reason:  "LocalNamespaceScanCompleted",
				Message: "Scanned 1 selected resource kinds in " + namespace + ".",
			}},
		},
	}
}

func buildArchiveExpectation(t *testing.T, scan types.NamespaceScan) any {
	t.Helper()

	contents, summary, err := archivepkg.BuildScanArchive(scan)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	reader, err := zip.NewReader(bytes.NewReader(contents), int64(len(contents)))
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(reader.File, func(i, j int) bool {
		return reader.File[i].Name < reader.File[j].Name
	})

	files := make([]map[string]any, 0, len(reader.File))
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		contentBytes := mustReadAll(t, handle)
		_ = handle.Close()

		entry := map[string]any{
			"path": file.Name,
		}
		if filepath.Ext(file.Name) == ".yaml" {
			var yamlValue any
			if err := yaml.Unmarshal(contentBytes, &yamlValue); err != nil {
				t.Fatal(err)
			}
			entry["type"] = "yaml"
			entry["content"] = normalizeValue(yamlValue)
		} else {
			entry["type"] = "text"
			entry["content"] = string(contentBytes)
		}
		files = append(files, entry)
	}

	return normalizeValue(map[string]any{
		"archiveName":   summary.ArchiveName,
		"manifestCount": summary.ManifestCount,
		"warningCount":  summary.WarningCount,
		"files":         files,
	})
}

func readFixtureConfig(t *testing.T, fixtureDir string) fixtureConfig {
	t.Helper()
	filePath := filepath.Join(fixtureDir, "fixture.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fixtureConfig{}
		}
		t.Fatal(err)
	}
	var config fixtureConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	return config
}

func readClassificationJSON(t *testing.T, filePath string) types.ResourceClassification {
	t.Helper()
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	var value types.ResourceClassification
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func readResourceYAML(t *testing.T, filePath string) types.ResourceObject {
	t.Helper()
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	var value types.ResourceObject
	if err := yaml.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func readNormalizedYAML(t *testing.T, filePath string) any {
	t.Helper()
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := yaml.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return normalizeValue(value)
}

func readJSONAny(t *testing.T, filePath string) any {
	t.Helper()
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return normalizeValue(value)
}

func normalizeValue(value any) any {
	return normalizeReflect(reflect.ValueOf(value))
}

func mustReadAll(t *testing.T, handle interface{ Read([]byte) (int, error) }) []byte {
	t.Helper()
	data, err := io.ReadAll(readerAdapter{handle})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type readerAdapter struct {
	reader interface{ Read([]byte) (int, error) }
}

func (r readerAdapter) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func apiGroup(value string) string {
	for i := 0; i < len(value); i++ {
		if value[i] == '/' {
			return value[:i]
		}
	}
	return ""
}

func apiVersion(value string) string {
	for i := len(value) - 1; i >= 0; i-- {
		if value[i] == '/' {
			return value[i+1:]
		}
	}
	return value
}

func normalizeReflect(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}

	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		return normalizeReflect(value.Elem())
	case reflect.Map:
		if value.IsNil() {
			return nil
		}
		normalized := make(map[string]any, value.Len())
		for _, key := range value.MapKeys() {
			normalized[key.String()] = normalizeReflect(value.MapIndex(key))
		}
		return normalized
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice && value.IsNil() {
			return nil
		}
		normalized := make([]any, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			normalized = append(normalized, normalizeReflect(value.Index(i)))
		}
		return normalized
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(value.Uint())
	case reflect.Float32, reflect.Float64:
		return value.Convert(reflect.TypeOf(float64(0))).Interface()
	default:
		return value.Interface()
	}
}
