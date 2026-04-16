---
title: Command Reference
description: >-
  Full command reference for the scrubctl CLI.
---

# Command Reference

Complete reference for every `scrubctl` subcommand, flag, and supported resource kind. The cluster-facing subcommands (`scan`, `export`, `generate argocd`) read your active kubeconfig the same way `kubectl` or `oc` do. The `scrub` subcommand and stdin pipe mode work on local YAML only, so they need no cluster access and no kubeconfig.

For installation instructions, quick start examples, and the embedded demo, see <a href="{{ '/' | relative_url }}"><kbd>DOCS HOME</kbd></a>.

## Usage

```sh
scrubctl --help
```

```text
Sanitize live manifests and generate GitOps export artifacts

Usage:
  scrubctl [flags]
  scrubctl [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  export      Export a namespace scan as a ZIP archive
  generate    Generate GitOps manifests
  help        Help about any command
  scan        Scan a namespace and print the classification table
  scrub       Scrub a single YAML resource from file or stdin
  version     Print the CLI version

Flags:
      --config string            Path to a config file for default flag values
      --context string           Kubeconfig context to use
      --exclude-kinds string     Comma-separated curated kinds or registry keys to exclude
  -h, --help                     help for scrubctl
      --include-kinds string     Comma-separated curated kinds or registry keys to include
      --kubeconfig string        Path to the kubeconfig file
      --log-level string         Log level (default "info")
  -n, --namespace string         Target namespace
  -q, --quiet                    Suppress non-essential output
      --secret-handling string   Secret handling mode: redact, omit, or include (default "redact")

Use "scrubctl [command] --help" for more information about a command.
```

## Commands

### Pipe a live resource

When invoked with no subcommand and YAML on stdin, `scrubctl` scrubs the resource directly:

```sh
oc get deploy/<name> -n <namespace> -o yaml | scrubctl
kubectl get deploy/<name> -n <namespace> -o yaml | scrubctl
```

### Scrub a resource file

```sh
scrubctl scrub -f deployment.yaml
scrubctl scrub -f resource.json
scrubctl scrub < resource.yaml
```

### Scan a namespace

Prints a classification table of every resource in the namespace:

```sh
scrubctl scan <namespace>
```

### Export a namespace to a ZIP archive

```sh
scrubctl export <namespace> -o <dir>
```

The archive contains `README.md`, optional `WARNINGS.md`, and manifest files organized under `manifests/include/`, `manifests/cleanup/`, and `manifests/review/`.

### Generate an Argo CD Application

```sh
scrubctl generate argocd <namespace> \
  --repo-url https://github.com/example/repo.git \
  --revision main \
  --path manifests/overlays/install
```

Example output:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  labels:
    app.kubernetes.io/managed-by: <application-name>
    gitops-export/namespace: <namespace>
    gitops-export/scanned-at: <timestamp>
  name: <application-name>
  namespace: openshift-gitops
spec:
  destination:
    namespace: <namespace>
    server: https://kubernetes.default.svc
  project: default
  source:
    directory:
      recurse: true
    path: manifests/overlays/install
    repoURL: https://github.com/example/repo.git
    targetRevision: main
```

### Print version

```sh
scrubctl version
```

## Global flags

- `--config` — path to a config file for default flag values (see [Config file](#config-file))
- `--kubeconfig` — path to the kubeconfig file
- `--context` — kubeconfig context to use
- `-n, --namespace` — target namespace
- `--secret-handling redact|omit|include` — how to handle Secret values (default: `redact`)
- `--include-kinds` — comma-separated curated kinds to include
- `--exclude-kinds` — comma-separated curated kinds to exclude
- `-q, --quiet` — suppress non-essential output
- `--log-level` — log level (default: `info`)

If you do not pass a namespace argument, the CLI falls back to `-n/--namespace` and then the active kubeconfig context namespace.

## Config file

You can define default `includeKinds` and `excludeKinds` in a YAML config file and pass it with `--config`:

```yaml
# scrubctl.yaml
includeKinds:
  - Deployment
  - Service
  - ConfigMap

excludeKinds:
  - Secret
  - Route
```

```sh
scrubctl scan my-app --config scrubctl.yaml
```

Config is only loaded when `--config` is explicitly provided. There is no auto-discovery from the current directory or home directory.

CLI flags always take precedence over config values. If both `--include-kinds` and `includeKinds` are set, the CLI flag wins:

```sh
# Uses Deployment,Service from CLI, ignores config includeKinds
scrubctl scan my-app --config scrubctl.yaml --include-kinds Deployment,Service
```

## Resource scope

### Curated set (scan / export)

The `scan`, `export`, and `generate argocd` subcommands work with a curated set of namespaced resource kinds:

- Kubernetes: Deployment, StatefulSet, DaemonSet, Job, CronJob, Service, Secret, ConfigMap, PersistentVolumeClaim, NetworkPolicy, HorizontalPodAutoscaler, Ingress, Role, RoleBinding, ServiceAccount, LimitRange, PodDisruptionBudget, ResourceQuota
- OpenShift: Route, BuildConfig, ImageStream, ImageStreamTag

Kinds outside that set are excluded with `kind not in curated resource set`. Use `--include-kinds` and `--exclude-kinds` to filter the curated set within these boundaries.

### Broad set (scrub / stdin)

The `scrub` subcommand and stdin pipe mode accept a broader supported set of ~32 resource kinds, since you are explicitly providing the input. In addition to the curated kinds above, direct scrub supports cluster-scoped kinds (ClusterRole, ClusterRoleBinding, Namespace, PersistentVolume, StorageClass), runtime kinds (Pod, ReplicaSet, Endpoints), and infrastructure kinds (CustomResourceDefinition, ValidatingWebhookConfiguration, MutatingWebhookConfiguration). Resources that would be excluded in a scan context (e.g. controller-owned, runtime-generated) are instead classified as `review` and still sanitized and output.

Kinds outside the supported set are rejected with `kind not supported for direct scrub`.

## OpenShift and oc

`scrubctl` works naturally alongside `oc`. Pipe any resource fetched with `oc get` directly:

```sh
oc get deploy/<name> -n <namespace> -o yaml | scrubctl
oc get route/<name> -n <namespace> -o yaml | scrubctl
```

Use `-n` or `--namespace` to target a namespace directly when running `scan` or `export` against an OpenShift cluster with an active `oc` session. OpenShift resource kinds (Route, BuildConfig, ImageStream, ImageStreamTag) are first-class and handled identically to standard Kubernetes kinds.

## Local development

Use the Make targets from the repo root:

| Target | What it does |
|--------|-------------|
| `make build` | Compiles `scrubctl` to `./bin/scrubctl` |
| `make install` | Runs `go install ./cmd/scrubctl`; binary lands in `$(go env GOBIN)` if set, otherwise `$(go env GOPATH)/bin` |
| `make test` | Runs Go unit tests |
