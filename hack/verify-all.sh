#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GIT_STATUS="$(git status --porcelain)"
GIT_DIFF="$(git diff)"
GOFMT_OUT="$(find ./{pkg,cmd}/ -name '*.go' -not -path './go/vendor/*' -not -path './pkg/apis/picchu/v1alpha1/zz_*.go' -exec gofmt -l -s {} \;)"

if [[ -n "${GOFMT_OUT}" || -n "${GIT_STATUS}" ]]; then
    if [[ -n "${GOFMT_OUT}" ]]; then
      echo "${GOFMT_OUT}"
      echo "** gofmt FAILED"
    fi
    if [[ -n "${GIT_STATUS}" ]]; then
      echo "${GIT_STATUS}"
      echo "** git status shows changes"
      echo "${GIT_DIFF}"
    fi
  exit 1
fi
