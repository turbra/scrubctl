---
title: scrubctl
description: >-
  Documentation home for the scrubctl CLI.
---

<div class="sc-badge-row">
  <a href="https://www.apache.org/licenses/LICENSE-2.0"><img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache--2.0-2C7A7B?style=flat-square" /></a>
</div>

A standalone Go CLI that scrubs Kubernetes and OpenShift manifests, scans namespaces, and exports GitOps-ready artifacts. Use it from the terminal or automation pipelines for inline scrubbing and clean GitOps export workflows.

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

Build from source (requires Go 1.24+):

```sh
go build -o scrubctl ./cmd/scrubctl
sudo mv scrubctl /usr/local/bin/
scrubctl version
```

Or install directly:

```sh
go install github.com/turbra/scrubctl/cmd/scrubctl@latest
```

`go install` writes the binary to `$(go env GOBIN)` if set, otherwise `$(go env GOPATH)/bin` (typically `~/go/bin`). It does not place `scrubctl` into `/usr/local/bin` or another system path automatically.

If that directory is not already on your `PATH`, add it before running `scrubctl`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
hash -r
scrubctl version
```

Tagged releases are published automatically when a `v*.*.*` tag is pushed. The release matrix follows the same primary platform targets that the `oc` client ships today:

- Linux: `amd64`, `arm64`, `ppc64le`, `s390x`
- macOS: `amd64`, `arm64`
- Windows: `amd64`

Linux and macOS release archives are published as `.tar.gz`; Windows archives are published as `.zip` on the [GitHub Releases page](https://github.com/turbra/scrubctl/releases).
Each tagged release also includes SBOM JSON files for the published archives.

### Install from a release archive

Download the archive matching your OS and architecture from the [GitHub Releases page](https://github.com/turbra/scrubctl/releases).

Linux:

```sh
tar -xzf scrubctl-<version>-linux-<arch>.tar.gz
sudo mv scrubctl /usr/local/bin/
scrubctl version
```

Windows:

```powershell
Expand-Archive scrubctl-<version>-windows-<arch>.zip -DestinationPath .
move .\scrubctl.exe $env:USERPROFILE\AppData\Local\Microsoft\WindowsApps\
scrubctl.exe version
```

## Quick Start

```sh
# Scrub a single resource file
scrubctl scrub -f deployment.yaml

# Pipe a live resource through scrubctl
oc get deploy/web -n my-app -o yaml | scrubctl

# Scan a namespace
scrubctl scan my-app

# Export a namespace as a ZIP archive
scrubctl export my-app -o ./out

# Generate an Argo CD Application manifest
scrubctl generate argocd my-app \
  --repo-url https://github.com/example/repo.git \
  --revision main \
  --path manifests/overlays/install
```

## Reference

- <a href="{{ '/cli.html' | relative_url }}"><kbd>COMMAND REFERENCE</kbd></a>
  for full command details, global flags, and examples
- <a href="{{ '/testing.html' | relative_url }}"><kbd>TESTING</kbd></a>
  for how fixtures, parity tests, and sanitization quality checks work

## Related

`scrubctl` shares classification and sanitization logic with the [GitOps Export](https://github.com/turbra/gitops-export-plugin) OpenShift console plugin. Both tools produce identical output for the same input — verified by shared golden test fixtures.

## Repository

- <a href="https://github.com/turbra/scrubctl"><kbd>REPOSITORY</kbd></a>
- <a href="https://github.com/turbra/scrubctl/blob/main/README.md"><kbd>README</kbd></a>
