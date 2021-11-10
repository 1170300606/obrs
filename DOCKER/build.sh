#!/usr/bin/env bash
set -ex

BINARY=chain_bft

rm -f $BINARY
cd ..

# init small bank database
make build
./build/$BINARY init-db --account-sum 1000 --dir ./build

make build-linux
cd DOCKER
cp ../build/$BINARY ./
cp -R ../build/smallbank.db ./

docker build -t "chain_bft:latest" .

rm -rf ./smallbank.db