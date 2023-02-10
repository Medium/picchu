#!/bin/bash

find ./vendor/istio.io -type f -exec grep 'protobuf_oneof' -l {} \; -exec perl -i -pe's/(protobuf_oneof.*)`$/$1 json:\"-\"`/g' {} \;