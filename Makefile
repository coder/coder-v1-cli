# Makefile for Coder CLI

.PHONY: clean build build/macos build/windows build/linux fmt lint gendocs test/go dev

PROJECT_ROOT := $(shell git rev-parse --show-toplevel)
MAKE_ROOT := $(shell pwd)

clean:
	rm -rf ./ci/bin

build: build/macos build/windows build/linux

build/macos:
	# requires darwin
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 ./ci/scripts/build.sh
build/windows:
	CGO_ENABLED=0 GOOS=windows ./ci/scripts/build.sh
build/linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ./ci/scripts/build.sh

fmt:
	go mod tidy
	gofmt -w -s .
	goimports -w "-local=$$(go list -m)" .

lint:
	golangci-lint run -c .golangci.yml

gendocs:
	rm -rf ./docs
	mkdir ./docs
	go run ./cmd/coder gen-docs ./docs

test/go:
	go test $$(go list ./... | grep -v pkg/tcli | grep -v ci/integration)

test/coverage:
	go test \
		-race \
		-covermode atomic \
		-coverprofile coverage \
		$$(go list ./... | grep -v pkg/tcli | grep -v ci/integration)

	goveralls -coverprofile=coverage -service=github

dev: build/linux
	@echo "removing project root binary if exists"
	-rm ./coder
	@echo "untarring..."
	@tar -xzf ./ci/bin/coder-cli-linux-amd64.tar.gz
	@echo "new dev binary ready"