#!/usr/bin/env bash
set -e
TAG="v1.3"

docker tag chain_bft:latest 10.77.70.82:4433/chain_bft:$TAG
docker push 10.77.70.82:4433/chain_bft:$TAG
