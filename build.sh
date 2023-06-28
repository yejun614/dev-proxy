#!/bin/bash

exe=dev-proxy

mkdir build

for val in $(go tool dist list)
do
  os=$(echo $val | cut -d "/" -f 1)
  arch=$(echo $val | cut -d "/" -f 2)
  echo os = $os, arch = $arch

  GOOS=$os GOARCH=$arch go build -o ./build/${exe}-${os}-${arch} .
done
