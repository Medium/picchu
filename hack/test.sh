#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

export CGO_ENABLED=0

TARGETS=(
  ./cmd/...
  ./pkg/...
)

if [[ ${#@} -ne 0 ]]; then
  TARGETS=("$@")
fi

echo "Running tests:" "${TARGETS[@]}"

if [[ -n "${GOFLAGS:-}" ]]; then
  echo "GOFLAGS: ${GOFLAGS}"
fi

go test -installsuffix "static" -timeout 60s "${TARGETS[@]}"
echo "Success!"
