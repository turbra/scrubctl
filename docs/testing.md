# Testing

scrubctl uses fixture-based golden tests as its primary test strategy. The fixture files under `testdata/fixtures/` are test inputs and expected outputs only — they are not runtime data used by the scrubctl binary.

## Fixture directories

Each directory under `testdata/fixtures/` represents one resource scenario. There are currently 26 fixtures covering Deployments, StatefulSets, CronJobs, PVCs, Secrets, Services, Routes, and more.

Each fixture contains four files:

| File | Purpose |
|------|---------|
| `input.yaml` | A Kubernetes resource as it would appear from the cluster (with server-assigned fields, defaults, status, etc.) |
| `expected-classification.json` | The expected classification result: include, cleanup, review, or exclude, with the reason string |
| `expected-sanitized.yaml` | The expected sanitized resource after scrubctl strips defaults and server-assigned fields. `nil` for excluded resources. |
| `expected-archive.json` | The expected archive output structure (README, WARNINGS, manifest files) |

An optional `fixture.json` can override test defaults like `secretHandling`, `namespace`, or `scannedAt`.

## How fixture tests work

`TestFixturesMatchTSExpectations` in `internal/parity/parity_test.go` auto-discovers all fixture directories and runs each through the full pipeline:

```
input.yaml
   |
   v
classify (ClassifyCuratedResource)
   |--- compared to expected-classification.json
   v
sanitize (SanitizeResource)
   |--- compared to expected-sanitized.yaml
   v
archive (BuildScanArchive)
   '--- compared to expected-archive.json
```

Each comparison uses `go-cmp/cmp.Diff`. On failure, you get a unified diff showing exactly which fields differ between expected and actual output.

This validates that the classification, sanitization, and archive logic in the binary produces the expected output for each scenario. The fixture files themselves are inert test assets.

## Sanitization quality tests

`TestSanitizationQuality` in `internal/sanitize/sanitize_quality_test.go` is a separate cross-cutting test. It runs against all fixture inputs and checks universal invariants that should hold for every sanitized resource:

```
all fixture input.yaml files
   |
   v
sanitize_quality_test.go
   |
   v
cross-cutting checks:
 - no metadata.uid, resourceVersion, generation, creationTimestamp, managedFields, selfLink, ownerReferences
 - no forbidden annotation prefixes (pv.kubernetes.io/, volume.beta.kubernetes.io/, operator.openshift.io/, etc.)
 - no status field
 - no nested creationTimestamp: null anywhere in the tree
 - no empty securityContext: {} or affinity: {} maps
```

Fixtures classified as `exclude` are skipped. This test catches cleanup regressions that golden file comparison alone might miss — for example, a new fixture that accidentally includes `creationTimestamp: null` in both input and expected output.

## When to add a new fixture

Add a fixture when:

- Adding sanitization support for a new resource kind
- A sanitization bug is discovered (capture the failing input as a regression test)
- A new classification or archive behavior needs to be locked in
- A real cluster resource exposes cleanup gaps not covered by existing fixtures

Name fixtures descriptively: `cronjob-cleanup`, `pvc-annotated`, `statefulset-vct`, `secret-omit`.

## Fixtures vs unit tests

**Fixtures** test realistic end-to-end scenarios: a full resource goes through classify, sanitize, and archive, and the output is compared to golden files. Use fixtures when you care about the combined behavior of the pipeline on a real-shaped resource.

**Unit tests** test focused behavior in a single function or code path. Use unit tests for edge cases in helper functions, boundary conditions, or logic that doesn't need a full resource to exercise.

Both run with `go test ./...`.

## Updating expected outputs

When scrubctl behavior intentionally changes (new fields stripped, classification reason updated, etc.):

1. Run `go test ./internal/parity/ -v` to see which fixtures fail and what the diff looks like.
2. Verify the diff represents a true improvement, not an accidental regression.
3. Update `expected-sanitized.yaml`, `expected-classification.json`, and `expected-archive.json` to match the new output.
4. Run `go test ./...` to confirm everything passes, including the quality test.

When adding new fixtures, prefer inputs derived from real cluster resources — they catch real-world cleanup gaps that synthetic inputs miss. Strip any sensitive values before committing.
