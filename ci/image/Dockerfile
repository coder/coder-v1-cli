FROM golang:1

RUN apt-get update && apt-get install curl zip
ENV DENO_INSTALL /bin/deno
RUN curl -fsSL https://deno.land/x/install/install.sh | sh
ENV PATH $DENO_INSTALL/bin:$PATH

ENV GOFLAGS="-mod=readonly"
ENV CI=true

RUN go get golang.org/x/tools/cmd/goimports
RUN go get github.com/mattn/goveralls
RUN apt update && apt install grep
