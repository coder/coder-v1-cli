FROM golang:1.15

ENV GOFLAGS="-mod=readonly"
ENV CI=true

RUN go get golang.org/x/tools/cmd/goimports
RUN go get github.com/mattn/goveralls
RUN apt update && apt install grep zip tar
