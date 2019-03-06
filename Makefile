# Copyright © 2019 A Medium Corporation.
# Licensed under the Apache License, Version 2.0; see the NOTICE file.

PACKAGE := go.medium.engineering/picchu/pkg
API_PACKAGE := $(PACKAGE)/apis
GROUPS := picchu/v1alpha1
BOILERPLATE := hack/header.go.txt
GEN := zz_generated

.PHONY: all build generate

all: deps generate build

build:
	@mkdir -p build/_output/bin
	go build -o build/_output/bin/picchu ./cmd/manager

deps:
	dep ensure -v

generate: deepcopy defaulter openapi clientset lister informer

deepcopy: generators/deepcopy
	$< -i $(API_PACKAGE)/$(GROUPS) -O $(GEN).deepcopy -h $(BOILERPLATE)

defaulter: generators/defaulter
	$< -i $(API_PACKAGE)/$(GROUPS) -O $(GEN).defaults -h $(BOILERPLATE)

openapi: generators/openapi
	$< -i $(API_PACKAGE)/$(GROUPS) -O openapi -p $(PACKAGE)/openapi -h $(BOILERPLATE)

clientset: generators/client
	$< -p $(PACKAGE) --input-base $(API_PACKAGE) --input $(GROUPS) -n client -h $(BOILERPLATE)

lister: generators/lister
	$< -i $(API_PACKAGE)/$(GROUPS) -p $(PACKAGE)/client/listers -h $(BOILERPLATE)

informer: generators/informer clientset lister
	generators/informer -i $(API_PACKAGE)/$(GROUPS) -p $(PACKAGE)/client/informers --versioned-clientset-package $(PACKAGE)/client --listers-package $(PACKAGE)/client/listers -h $(BOILERPLATE)

generators/%: Gopkg.lock
	@mkdir -p generators
	go build -o $@ ./vendor/k8s.io/code-generator/cmd/$*-gen

generators/openapi: Gopkg.lock
	@mkdir -p generators
	go build -o generators/openapi ./vendor/k8s.io/kube-openapi/cmd/openapi-gen
