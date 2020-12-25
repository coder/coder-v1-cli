# Makefile for Coder CLI 

.PHONY: clean build build/macos build/windows build/linux

clean:
	rm -rf ./ci/bin

build: build/macos build/windows build/linux

build/macos:
	# requires darwin
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 ./ci/steps/build.ts
build/windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=386 ./ci/steps/build.ts
build/linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ./ci/steps/build.ts
