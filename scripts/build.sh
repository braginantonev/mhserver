#!/bin/bash
# Run in root of project tree

if [[ !(-e build) ]]; then
    mkdir build
fi

go build -C cmd/ -o main
mv cmd/main build/

cp -r scripts/ build/
cp -r sql/ build/

rm build/scripts/build.sh