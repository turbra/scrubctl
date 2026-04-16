---
title: Command Reference
description: >-
  Full command reference for the scrubctl CLI.
---

# Command Reference

`scrubctl` is a standalone Go CLI for namespace scan, resource classification, manifest sanitization, ZIP export, and Argo CD Application generation. It runs as a local binary — no OpenShift console required.

The cluster-facing subcommands (`scan`, `export`, `generate argocd`) read your active kubeconfig the same way `kubectl` or `oc` do. The `scrub` subcommand and stdin pipe mode work on local YAML only, so they need no cluster access and no kubeconfig.

## Demo

<div id="demo-player"></div>
<script>
  document.addEventListener('DOMContentLoaded', function () {
    AsciinemaPlayer.create(
      '{{ "/demo-scrubctl.cast" | relative_url }}',
      document.getElementById('demo-player'),
      { cols: 120, rows: 36, idleTimeLimit: 3, speed: 1.5, theme: 'asciinema', fit: 'width' }
    );
  });
</script>

## Install

Three steps, every time:

1. Build or download the `scrubctl` binary.
2. Place it in a directory on your `PATH`.
3. Verify with `scrubctl version`.

### Install from a release archive

> Not yet available — no tagged releases have been published. Use **Build from source** below until the first `v*.*.*` tag ships.

```sh
tar -xzf scrubctl-<version>-<os>-<arch>.tar.gz
sudo mv scrubctl /usr/local/bin/
scrubctl version
```

Download the archive matching your OS/arch from the [GitHub Releases page](https://github.com/turbra/scrubctl/releases) (Linux, macOS, Windows — amd64 and arm64). On Windows, extract the `.zip` and move `scrubctl.exe` into a directory on your `PATH`.

### Build from source

From a clone of this repository (requires Go 1.21+):

```sh
go build -o scrubctl ./cmd/scrubctl
sudo mv scrubctl /usr/local/bin/
scrubctl version
```

### Notes

- If `/usr/local/bin` is not on your `PATH`, run `echo $PATH` and either move `scrubctl` into a directory that is listed, or add its directory to your `PATH` (for example, `export PATH="$PATH:$HOME/.local/bin"` in `~/.bashrc` or `~/.zshrc`).
- `go install github.com/turbra/scrubctl/cmd/scrubctl@latest` places the binary in `$(go env GOBIN)` if set, otherwise `$(go env GOPATH)/bin` (typically `~/go/bin`). Add that directory to your `PATH`, or copy `scrubctl` from it into `/usr/local/bin/`.

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

### Quick examples

Five of the most common invocations at a glance. Each one maps to a full section under **Commands** below.

```sh
# Scrub a single resource file — no cluster access needed
scrubctl scrub -f deployment.yaml

# Pipe a live resource through scrubctl
oc get deploy/web -n my-app -o yaml | scrubctl

# Scan a namespace and print the classification table
scrubctl scan my-app

# Export a namespace as a ZIP archive into ./out
scrubctl export my-app -o ./out

# Generate an Argo CD Application manifest
scrubctl generate argocd my-app \
  --repo-url https://github.com/example/repo.git \
  --revision main \
  --path manifests/overlays/install
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

- `--kubeconfig` — path to the kubeconfig file
- `--context` — kubeconfig context to use
- `-n, --namespace` — target namespace
- `--secret-handling redact|omit|include` — how to handle Secret values (default: `redact`)
- `--include-kinds` — comma-separated curated kinds to include
- `--exclude-kinds` — comma-separated curated kinds to exclude
- `-q, --quiet` — suppress non-essential output
- `--log-level` — log level (default: `info`)

If you do not pass a namespace argument, the CLI falls back to `-n/--namespace` and then the active kubeconfig context namespace.

## Resource scope

`scrubctl` supports a curated set of namespaced resource kinds:

- Kubernetes: Deployment, StatefulSet, DaemonSet, Job, CronJob, Service, Secret, ConfigMap, PersistentVolumeClaim, NetworkPolicy, HorizontalPodAutoscaler, Role, RoleBinding, ServiceAccount
- OpenShift: Route, BuildConfig, ImageStream, ImageStreamTag

Kinds outside that set are excluded with `kind not in curated resource set`.

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
