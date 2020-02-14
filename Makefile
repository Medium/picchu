# Copyright © 2019 A Medium Corporation.
# Licensed under the Apache License, Version 2.0; see the NOTICE file.

DOMAIN := medium.engineering
PACKAGE := go.$(DOMAIN)/picchu/pkg
API_PACKAGE := $(PACKAGE)/apis
GROUPS := picchu/v1alpha1
BOILERPLATE := hack/header.go.txt
GEN := zz_generated

platform_temp = $(subst -, ,$(ARCH))
GOOS = $(word 1, $(platform_temp))
GOARCH = $(word 2, $(platform_temp))
GOROOT = $(shell go env GOROOT)

.PHONY: all build generate deepcopy defaulter openapi clientset crds ci test verify

all: deps generate build

build:
	@mkdir -p build/_output/bin
	go build -o build/_output/bin/picchu ./cmd/manager

docker:
	# https://github.com/operator-framework/operator-sdk/issues/1854#issuecomment-569285967
	GOROOT=$(GOROOT) operator-sdk build $(IMAGE)

deps:
	go mod tidy
	go mod vendor

generate: deepcopy defaulter openapi clientset

deepcopy:
	# https://github.com/operator-framework/operator-sdk/issues/1854#issuecomment-569285967
	GOROOT=$(GOROOT) operator-sdk generate k8s

defaulter: generators/defaulter
	$< -i $(API_PACKAGE)/$(GROUPS) -O $(GEN).defaults -h $(BOILERPLATE)

openapi: generators/openapi-gen
	$< -i $(API_PACKAGE)/$(GROUPS) -O $(GEN).openapi -p $(PACKAGE)/openapi -h $(BOILERPLATE)

clientset: generators/client
	$< -p $(PACKAGE) --input-base $(API_PACKAGE) --input $(GROUPS) -n client -h $(BOILERPLATE)

crds:
	operator-sdk generate crds

generators/%:
	@mkdir -p generators
	go build -o $@ k8s.io/code-generator/cmd/$*-gen

generators/openapi-gen:
	@mkdir -p generators
	go build -o ./generators/openapi-gen k8s.io/kube-openapi/cmd/openapi-gen

build-dirs:
	@mkdir -p _output/bin/$(GOOS)/$(GOARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(GOOS)/$(GOARCH) .go/go-build

test: build-dirs
	hack/test.sh

verify: crds
	hack/verify-all.sh

ci: all verify test

mocks:
	@mkdir -p generators
	go build -o ./generators/mockgen github.com/golang/mock/mockgen
	./generators/mockgen -destination=pkg/mocks/client.go -package=mocks sigs.k8s.io/controller-runtime/pkg/client Client
	./generators/mockgen -destination=pkg/prometheus/mocks/mock_promapi.go -package=mocks $(PACKAGE)/prometheus PromAPI
	./generators/mockgen -destination=pkg/controller/releasemanager/mock_deployment.go -package=releasemanager $(PACKAGE)/controller/releasemanager Deployment
	./generators/mockgen -destination=pkg/controller/releasemanager/mock_incarnations.go -package=releasemanager $(PACKAGE)/controller/releasemanager Incarnations
	./generators/mockgen -destination=pkg/controller/releasemanager/scaling/mocks/scalabletarget_mock.go -package=mocks $(PACKAGE)/controller/releasemanager/scaling ScalableTarget
	./generators/mockgen -destination=pkg/plan/mocks/plan_mock.go -package=mocks $(PACKAGE)/plan Plan
