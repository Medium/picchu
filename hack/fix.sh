#!/bin/bash

find ./vendor/istio.io -type f -exec grep 'protobuf_oneof' -l {} \; -exec perl -i -pe's/(protobuf_oneof.*)`$/$1 json:\"-\"`/g' {} \;
find ./vendor/istio.io -type f -exec grep ',omitempty"`$' -l {} \; -exec perl -i -pe 's/(,proto3" json:")([^,"]+)([^"]*)("?,)/$1 . lcfirst(join("", map { ucfirst($_) } split("_", $2))) . $3 . $4/ge;' {} \;
find ./vendor/istio.io -type f -exec grep 'h2_upgrade_policy,omitempty' -l {} \; -exec perl -i -pe 's/h2_upgrade_policy,omitempty/h2UpgradePolicy,omitempty/g' {} \;
