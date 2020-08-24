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
export GOROOT = $(shell go env GOROOT)
export GO111MODULE = on

.PHONY: all build generate deepcopy defaulter openapi clientset crds ci test verify

all: deps generate build
ci: all verify generate test
generate: deepcopy defaulter openapi clientset matcher

build:
	@mkdir -p build/_output/bin
	go build -o build/_output/bin/picchu ./cmd/manager
	go build -o build/_output/bin/picchu-webhook ./cmd/webhook

docker:
	# https://github.com/operator-framework/operator-sdk/issues/1854#issuecomment-569285967
	operator-sdk build $(IMAGE)
	docker build -t $(WEBHOOK_IMAGE) -f build/webhook.Dockerfile .

deps:
	go mod tidy
	go mod vendor

deepcopy:
	# https://github.com/operator-framework/operator-sdk/issues/1854#issuecomment-569285967
	operator-sdk generate k8s

defaulter: generators/defaulter
	$< -i $(API_PACKAGE)/$(GROUPS) -O $(GEN).defaults -h $(BOILERPLATE)

openapi: generators/openapi-gen
	$< -i $(API_PACKAGE)/$(GROUPS) -O $(GEN).openapi -p $(PACKAGE)/openapi -h $(BOILERPLATE)

clientset: generators/client
	$< -p $(PACKAGE) --input-base $(API_PACKAGE) --input $(GROUPS) -n client -h $(BOILERPLATE)

matcher: generators/matcher-gen
	$< -i github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1 -p go.medium.engineering/picchu/pkg/test/monitoring/v1

crds:
	operator-sdk generate crds

generators/%: go.sum
	@mkdir -p generators
	go build -o $@ k8s.io/code-generator/cmd/$*-gen

generators/matcher-gen: go.sum
	@mkdir -p generators
	go build -o $@ go.medium.engineering/kubernetes/cmd/matcher-gen

generators/openapi-gen: go.sum
	@mkdir -p generators
	go build -o ./generators/openapi-gen k8s.io/kube-openapi/cmd/openapi-gen

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
