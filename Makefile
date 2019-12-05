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

generate:
	operator-sdk generate k8s
	operator-sdk generate openapi

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
