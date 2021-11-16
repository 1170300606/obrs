#!/usr/bin/env bash
set -e
TAG="v1.1"
docker build -t "chain_bft:latest" .

docker tag chain_bft:latest 10.77.70.135:4433/chain_bft:$TAG
docker push 10.77.70.135:4433/chain_bft:$TAG
