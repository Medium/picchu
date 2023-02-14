#!/bin/bash
set -x
mkdir -p resources
VERSION=v3.1.0
UNAME=$(uname -s)
if [ ${UNAME} == "Linux" ]
then
    BINARY=yq_linux_amd64
elif [ ${UNAME} == "Darwin" ]
then 
    BINARY=yq_darwin_amd64
else
    echo "Unsupported platform ${UNAME}"
    exit 1
fi

which yq 
if [ $? != 0 ]
then
    wget https://github.com/mikefarah/yq/releases/download/${VERSION}/${BINARY}.tar.gz -O - |\
  tar xz && mv ${BINARY} yq && export PATH=$PATH:$PWD
fi

pushd resources
kustomize build ../config/crd | csplit - '/^---$/' {4} 
for files in $(ls xx*)
    do new_path=$(cat $files | yq '.metadata.annotations."config.kubernetes.io/origin"'|cut -d: -f2|cut -d \\ -f1|xargs)
    echo "Copying $files --> $new_path"
    #cp $files ../config/crd/finalized/${new_path}
    mv $files ../config/crd/${new_path}
    
done
popd 
rm -rf resources