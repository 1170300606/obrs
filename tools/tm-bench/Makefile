DIST_DIRS := find * -type d -exec

all: build test install

########################################
###  Build

build:
	@go build

install:
	@go install

test:
	@go test -race

########################################
### Docker

build-docker:
	rm -f ./tm-bench
	docker run -it --rm -v "$(PWD)/../../:/go/src/github.com/tendermint/tendermint" -w "/go/src/github.com/tendermint/tendermint/tools/tm-bench" -e "CGO_ENABLED=0" golang:alpine go build -ldflags "-s -w" -o tm-bench
	docker build -t "tendermint/bench" .

clean:
	rm -f ./tm-bench
	rm -rf ./dist

# To avoid unintended conflicts with file names, always add to .PHONY
# unless there is a reason not to.
# https://www.gnu.org/software/make/manual/html_node/Phony-Targets.html
.PHONY: build install test docker clean
