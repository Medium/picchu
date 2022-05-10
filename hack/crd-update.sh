#!/bin/bash

go get sigs.k8s.io/controller-tools/cmd/controller-gen@${1}
go mod tidy
go mod vendor
for crd in $(find ./deploy/crds -type f); do cp $crd ${crd%.yaml}_crd.yaml; done
controller-gen +crd:allowDangerousTypes=true,crdVersions=v1beta1 paths=./pkg/... output:crd:dir=./deploy/crds output:stdout