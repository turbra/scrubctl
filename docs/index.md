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

Build from source (requires Go 1.21+):

```sh
go build -o scrubctl ./cmd/scrubctl
sudo mv scrubctl /usr/local/bin/
scrubctl version
```

Or install directly:

```sh
go install github.com/turbra/scrubctl/cmd/scrubctl@latest
```

Release archives for Linux, macOS, and Windows are available on the [GitHub Releases page](https://github.com/turbra/scrubctl/releases).

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
