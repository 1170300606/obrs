#!/usr/bin/make -f
OUTPUT?=build/chain_bft

build-linux:
	GOOS=linux GOARCH=amd64 $(MAKE) build

build:
	go build -o $(OUTPUT) ./cmd/

test:
	make build
	./build/chain_bft run

.PHONY: build-linux build test
