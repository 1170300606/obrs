#!/usr/bin/env bash
set -e

BINARY=chain_bft

rm -f $BINARY
cd ..
make build-linux
cd DOCKER
cp ../build/$BINARY ./

docker build -t "chain_bft:latest" .
