#!/bin/bash
# build for window: build.sh windows
# default linux
#./gox -build-toolchain

set -e
cd $(dirname $0)

dest_file="pproxy"

DEST_OS=$1
if [ "$DEST_OS" == "windows" ];then
  export GOOS=windows 
  export GOARCH=386
  dest_file="pproxy.exe"
fi

go build -o $dest_file -ldflags "-s -w"  proxy_main.go 
zip -r res.zip res
cat res.zip>> $dest_file
zip -A $dest_file
mkdir -p dest/
mv $dest_file dest/
rm res.zip

echo "all finish"