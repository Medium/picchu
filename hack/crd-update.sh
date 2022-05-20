#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

go install sigs.k8s.io/controller-tools/cmd/controller-gen@${1}
go mod tidy
go mod vendor
#https://github.com/istio/api/issues/1482
find ./vendor/istio.io -type f -exec grep 'protobuf_oneof' -l {} \; -exec perl -i -pe's/"`$/" json:\"-\"`/g' {} \;
#Using controller-gen to allow float64 type, no current flag for operator-sdk
controller-gen --version
controller-gen +crd:allowDangerousTypes=true,crdVersions=v1,ignoreUnexportedFields=true paths=./pkg/... output:crd:dir=./deploy/crds output:stdout
for crd in $(find ./deploy/crds -type f|grep -v "_crd.yaml$"); do echo $crd; mv $crd ${crd%.yaml}_crd.yaml; done