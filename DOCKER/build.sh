#!/usr/bin/env bash
set -e

BINARY=chain_bft

rm -f $BINARY

cp ../build/$BINARY ./

sudo docker build -t "chain_bft:latest" .