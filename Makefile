# Copyright © 2019 A Medium Corporation.
# Licensed under the Apache License, Version 2.0; see the NOTICE file.

DOMAIN := medium.engineering
PACKAGE := go.$(DOMAIN)/picchu/pkg
API_PACKAGE := $(PACKAGE)/apis
GROUPS := picchu/v1alpha1
BOILERPLATE := hack/header.go.txt
GEN := zz_generated
OPERATOR_SDK_VERSION := v0.18.0
CONTROLLER_GEN_VERSION := v0.4.1

platform_temp = $(subst -, ,$(ARCH))
GOOS = $(word 1, $(platform_temp))
GOARCH = $(word 2, $(platform_temp))
export GOROOT = $(shell go env GOROOT)
export GO111MODULE = on

OPERATOR_SDK_PLATFORM := unknown

UNAME := $(shell uname -s)
ifeq ($(UNAME),Linux)
	OPERATOR_SDK_PLATFORM = linux-gnu
endif
ifeq ($(UNAME),Darwin)
	OPERATOR_SDK_PLATFORM = apple-darwin
endif

.PHONY: all build generate deepcopy defaulter openapi clientset crds ci test verify

all: deps generate build
ci: all verify generate test
generate: deepcopy defaulter openapi clientset matcher

build:
	@mkdir -p build/_output/bin
	go build -o build/_output/bin/picchu ./cmd/manager
	go build -o build/_output/bin/picchu-webhook ./cmd/webhook

docker: generators/operator-sdk
	# https://github.com/operator-framework/operator-sdk/issues/1854#issuecomment-569285967
	$< build $(IMAGE)
	docker build -t $(WEBHOOK_IMAGE) -f build/webhook.Dockerfile .

deps:
	go mod tidy
	go mod vendor

deepcopy: generators/operator-sdk
	# https://github.com/operator-framework/operator-sdk/issues/1854#issuecomment-569285967
	$< generate k8s

defaulter: generators/defaulter
	$< -i $(API_PACKAGE)/$(GROUPS) -O $(GEN).defaults -h $(BOILERPLATE)

openapi: generators/openapi-gen
	$< -i $(API_PACKAGE)/$(GROUPS) -O $(GEN).openapi -p $(PACKAGE)/openapi -h $(BOILERPLATE)

clientset: generators/client
	$< -p $(PACKAGE) --input-base $(API_PACKAGE) --input $(GROUPS) -n client -h $(BOILERPLATE)

matcher: generators/matcher-gen
	$< -i github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1 -p go.medium.engineering/picchu/pkg/test/monitoring/v1

crds: generators/operator-sdk
	hack/crd-update.sh $(CONTROLLER_GEN_VERSION)

generators/%: go.sum
	@mkdir -p generators
	go build -o $@ k8s.io/code-generator/cmd/$*-gen

generators/matcher-gen: go.sum
	@mkdir -p generators
	go build -o $@ go.medium.engineering/kubernetes/cmd/matcher-gen

generators/openapi-gen: go.sum
	@mkdir -p generators
	go build -o ./generators/openapi-gen k8s.io/kube-openapi/cmd/openapi-gen

generators/operator-sdk:
	@mkdir -p generators
	curl -L https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk-$(OPERATOR_SDK_VERSION)-x86_64-$(OPERATOR_SDK_PLATFORM) -o $@
	chmod +x generators/operator-sdk

build-dirs:
	@mkdir -p _output/bin/$(GOOS)/$(GOARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(GOOS)/$(GOARCH) .go/go-build

test: build-dirs
	hack/test.sh

verify: crds
	hack/verify-all.sh

fix:
	hack/fix-all.sh

mocks: go.sum
	@mkdir -p generators
	go get github.com/golang/mock/mockgen
	mockgen -destination=pkg/mocks/client.go -package=mocks sigs.k8s.io/controller-runtime/pkg/client Client
	mockgen -destination=pkg/prometheus/mocks/mock_promapi.go -package=mocks $(PACKAGE)/prometheus PromAPI
	mockgen -destination=pkg/controller/releasemanager/mock_deployment.go -package=releasemanager $(PACKAGE)/controller/releasemanager Deployment
	mockgen -destination=pkg/controller/releasemanager/mock_incarnations.go -package=releasemanager $(PACKAGE)/controller/releasemanager Incarnations
	mockgen -destination=pkg/controller/releasemanager/scaling/mocks/scalabletarget_mock.go -package=mocks $(PACKAGE)/controller/releasemanager/scaling ScalableTarget
	mockgen -destination=pkg/plan/mocks/plan_mock.go -package=mocks $(PACKAGE)/plan Plan
