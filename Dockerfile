ARG VERSION

FROM golang:$VERSION-alpine AS builder

ENV CI=true

RUN apk update && apk upgrade && apk --no-cache add curl

# Install latest gcc
RUN apk add build-base

# Copy local source
COPY . $GOPATH/src/github.com/alexsniffin/gosd.git/
WORKDIR $GOPATH/src/github.com/alexsniffin/gosd.git/

# Pull dependencies
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Run quality check
RUN golangci-lint run --timeout 10m0s -v --build-tags mus -c .golangci.yml \
    && go test -tags musl

# Build binary
RUN GOOS=linux go build -a .