#!/bin/bash
go get sigs.k8s.io/controller-tools/cmd/controller-gen@${1}
go mod tidy
go mod vendor
controller-gen +crd:allowDangerousTypes=true,crdVersions=v1beta1 paths=./pkg/... output:crd:dir=./deploy/crds output:stdout
for crd in $(find ./deploy/crds -type f|grep -v "_crd.yaml$"); do echo $crd; mv $crd ${crd%.yaml}_crd.yaml; done