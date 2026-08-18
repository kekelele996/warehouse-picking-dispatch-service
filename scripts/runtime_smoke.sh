#!/usr/bin/env bash
set -euo pipefail

cleanup() {
  docker rm -f warehouse-runtime-smoke >/dev/null 2>&1 || true
}

trap cleanup EXIT INT TERM
cleanup
docker build -f benzhi.Dockerfile -t warehouse-runtime-smoke:local .
docker run --rm --name warehouse-runtime-smoke -p 18068:8080 warehouse-runtime-smoke:local
