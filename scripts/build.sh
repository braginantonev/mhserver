#!/bin/bash
# Run in root of project tree

export VERSION=${VERSION:-$(git describe --tags --dirty --always | sed -e "s/^v//g")}

if [[ !(-e build) ]]; then
    mkdir build
else
    rm -rf build/*
fi

go build -C cmd/ -o ../build/mhserver -ldflags="-s -w -X=github.com/braginantonev/version.Version=${VERSION}"
mv cmd/mhserver build/

cp -r scripts/ build/scripts/
cp -r sql/ build/
cp mhserver.service build/

cd build

rm scripts/build.sh

tar -czvf mhserver.tar.gz *