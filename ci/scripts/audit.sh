#!/bin/bash -eux

export cwd=$(pwd)

pushd $cwd/dp-search-builder
  make audit
popd