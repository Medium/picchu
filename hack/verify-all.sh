#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GIT_STATUS="$(git status --porcelain)"
GOFMT_OUT="$(find ./{pkg,cmd}/ -name '*.go' -not -path './go/vendor/*' -not -name 'zz_generated.*' -exec gofmt -l -s -w {} \;)"

if [[ -n "${GOFMT_OUT}" || -n "${GIT_STATUS}" ]]; then
    if [[ -n "${GOFMT_OUT}" ]]; then
      echo "${GOFMT_OUT}"
      echo "** gofmt FAILED"
    fi
    if [[ -n "${GIT_STATUS}" ]]; then
      echo "${GIT_STATUS}"
      echo "** git status shows changes"
    fi
  exit 1
fi
