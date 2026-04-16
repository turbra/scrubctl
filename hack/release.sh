#!/usr/bin/env bash
set -euo pipefail

go test ./...
goreleaser release --clean
