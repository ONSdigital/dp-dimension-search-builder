#!/bin/bash -eux

export cwd=$(pwd)

pushd $cwd/dp-dimension-search-builder
  make audit
popd