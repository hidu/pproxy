#!/bin/bash
# build for window: build.sh windows
# default linux
#./gox -build-toolchain

set -e
cd $(dirname $0)

#export GOPATH=`readlink -f Godeps/_workspace`:$GOPATH

export GO15VENDOREXPERIMENT=1

go install