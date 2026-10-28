# autopi CLI — build targets. Requires a Go toolchain (build-time only).
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: all build test vet fmt clean contract-docs

contract-docs:
	python3 ../ci/cli/contracts/helpdocs.py --check

## all: cross-compile every platform into the skill's bin directory
all: build

## build: produce the committed static binaries
build:
	../ci/cli/build.sh $(VERSION)

## test: run unit tests
test:
	go test ./...

## vet: static analysis
vet:
	go vet ./...

## fmt: format sources
fmt:
	gofmt -w .

## clean: remove any local scratch build output
clean:
	rm -rf dist
