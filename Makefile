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

.PHONY: all build generate deepcopy defaulter openapi clientset crds ci test verify

all: deps generate build

build:
	@mkdir -p build/_output/bin
	go build -o build/_output/bin/picchu ./cmd/manager

deps:
	go mod tidy -v

generate: deepcopy defaulter openapi clientset

deepcopy: generators/deepcopy
	$< -i $(API_PACKAGE)/$(GROUPS) -O $(GEN).deepcopy -h $(BOILERPLATE)

defaulter: generators/defaulter
	$< -i $(API_PACKAGE)/$(GROUPS) -O $(GEN).defaults -h $(BOILERPLATE)

openapi: generators/openapi
	$< -i $(API_PACKAGE)/$(GROUPS) -O openapi -p $(PACKAGE)/openapi -h $(BOILERPLATE)

clientset: generators/client
	$< -p $(PACKAGE) --input-base $(API_PACKAGE) --input $(GROUPS) -n client -h $(BOILERPLATE)

crds: generators/crd
	@mkdir -p generated_crds
	$< generate --output-dir generated_crds --domain $(DOMAIN)

generators/%:
	@mkdir -p generators
	go build -o $@ ./vendor/k8s.io/code-generator/cmd/$*-gen

generators/openapi:
	@mkdir -p generators
	go build -o generators/openapi ./vendor/k8s.io/kube-openapi/cmd/openapi-gen

generators/crd:
	go get sigs.k8s.io/controller-tools/cmd/crd
	cp $(shell which crd) generators/crd

build-dirs:
	@mkdir -p _output/bin/$(GOOS)/$(GOARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(GOOS)/$(GOARCH) .go/go-build

test: build-dirs
	hack/test.sh

verifiy:
	hack/verify-all.sh

ci: all verify test

mocks:
	go get github.com/golang/mock/mockgen
	mockgen -destination=pkg/controller/releasemanager/mock_deployment.go -package=releasemanager $(PACKAGE)/controller/releasemanager Deployment
	mockgen -destination=pkg/controller/releasemanager/mock_incarnations.go -package=releasemanager $(PACKAGE)/controller/releasemanager Incarnations
	mockgen -destination=pkg/controller/releasemanager/scaling/mocks/scalabletarget_mock.go -package=mocks $(PACKAGE)/controller/releasemanager/scaling ScalableTarget
	mockgen -destination=pkg/plan/mocks/plan_mock.go -package=mocks $(PACKAGE)/plan Plan
