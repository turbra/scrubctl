#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

described="$(git -C "${REPO_ROOT}" describe --tags --match 'v[0-9]*' --dirty --always 2>/dev/null || true)"

if [[ -z "${described}" ]]; then
  echo "0.0.0"
  exit 0
fi

if [[ "${described}" =~ ^v?([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
  echo "${BASH_REMATCH[1]}"
  exit 0
fi

if [[ "${described}" =~ ^v?([0-9]+\.[0-9]+\.[0-9]+)-([0-9]+)-g([0-9a-f]+)(-dirty)?$ ]]; then
  suffix="${BASH_REMATCH[3]}"
  if [[ -n "${BASH_REMATCH[4]:-}" ]]; then
    suffix="${suffix}.dirty"
  fi
  echo "${BASH_REMATCH[1]}-dev.${BASH_REMATCH[2]}+${suffix}"
  exit 0
fi

if [[ "${described}" =~ ^([0-9a-f]+)(-dirty)?$ ]]; then
  suffix="${BASH_REMATCH[1]}"
  if [[ -n "${BASH_REMATCH[2]:-}" ]]; then
    suffix="${suffix}.dirty"
  fi
  echo "0.0.0-dev+${suffix}"
  exit 0
fi

echo "0.0.0"
