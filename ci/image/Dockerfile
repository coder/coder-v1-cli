FROM golang:1.12

ENV GOFLAGS="-mod=readonly"
ENV CI=true

RUN go get golang.org/x/tools/cmd/goimports
RUN go get github.com/mattn/goveralls
RUN apt-get update && DEBIAN_FRONTEND="noninteractive" apt-get install -y \
  grep \
  zip \
  tar
