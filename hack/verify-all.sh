#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -x

GIT_STATUS="$(git status --porcelain)"
GOFMT_OUT="$(find ./{pkg,cmd}/ -name '*.go' -not -path './go/vendor/*' -exec gofmt -l -s {} \;)"

if [[ -n "${GOFMT_OUT}" || -n "${GIT_STATUS}" ]]; then
    if [[ -n "${GOFMT_OUT}" ]]; then
      echo "${GOFMT_OUT}"
      echo "** gofmt FAILED"
      git diff ${GOFMT_OUT}
    fi
    if [[ -n "${GIT_STATUS}" ]]; then
      echo "${GIT_STATUS}"
      echo "** git status shows changes"
    fi
  exit 1
fi
