#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-dimension-search-builder
  make build && cp build/dp-dimension-search-builder $cwd/build
  cp Dockerfile.concourse $cwd/build
popd
