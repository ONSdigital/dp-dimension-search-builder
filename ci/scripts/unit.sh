#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-search-builder
  make test
popd
