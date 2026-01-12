#!/bin/bash
# Run in root of project tree

if [[ !(-e build) ]]; then
    mkdir build
else
    rm -rf build/*
fi

go build -C cmd/ -o mhserver
mv cmd/mhserver build/

cp -r scripts/* build/
cp -r sql/ build/
cp mhserver.service build/

cd build

rm build.sh

tar -czvf mhserver.tar.gz *