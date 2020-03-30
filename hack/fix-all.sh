#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

find ./{pkg,cmd} -name '*.go' -not -path './go/vendor/*' -exec gofmt -l -w {} \;
