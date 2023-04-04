#!/bin/bash

mkdir -p deploy

which yq || pip install yq

pushd deploy
kustomize build ../config/default | csplit -k - '/^---$/' {30} 
for files in $(ls xx*)
    do 
        type=$(cat $files | yq -r '.kind'|  awk '{print tolower($0)}');
        cat $files >> ${type}.yaml
        echo "Created ${type}.yaml"
done    
rm -rf xx*
popd 
