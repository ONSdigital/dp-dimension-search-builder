#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-search-builder
  make build && cp build/dp-search-builder $cwd/build
  cp Dockerfile.concourse $cwd/build
popd
