#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")"
tag="${1:-main}"
if [ "$#" -gt 0 ]; then shift; fi
exec go run ./cmd/horizon build --tag "$tag" "$@"
