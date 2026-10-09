#!/bin/bash
# Run in root of project tree

export VERSION=${VERSION:-$(git describe --tags --dirty --always | sed -e "s/^v//g")}

if [[ !(-e build) ]]; then
    mkdir build
else
    rm -rf build/*
fi

go build -C cmd/ -o ../build/mhserver -ldflags="-s -w -X=github.com/braginantonev/mhserver/version.Version=${VERSION}"

cp scripts/* build

cp -r config build/config

cd build

rm build.sh

tar -czvf mhserver.tar.gz *