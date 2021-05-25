OUTPUT?=build/bft


build:
	rm -rf $(OUTPUT)
	go build -o $(OUTPUT) ./cmd/

.PHONY: build