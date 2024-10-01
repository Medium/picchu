#!/bin/bash

mkdir -p resources

which yq || pip install yq

pushd resources
kustomize build ../config/crd | csplit - '/^---$/' {4} 
for files in $(ls xx*)
    do new_path=$(cat $files | yq '.metadata.annotations."config.kubernetes.io/origin"'|cut -d: -f2|cut -d \\ -f1|xargs)
    echo "Copying $files --> $new_path"
    mv $files ../config/crd/${new_path}
    
done
popd 
rm -rf resources