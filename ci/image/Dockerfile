FROM golang:1

ENV GOFLAGS="-mod=readonly"
ENV CI=true

RUN go get golang.org/x/tools/cmd/goimports
RUN go get golang.org/x/lint/golint
RUN go get github.com/mattn/goveralls