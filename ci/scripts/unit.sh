#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-dimension-search-builder
  make test
popd
