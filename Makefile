.PHONY: all default test lint build install tidy fix
default: all

test:
	GOFLAGS= go test -v -race ./...

tidy:
	go mod tidy

fix:
	go fix ./...
	gofmt -w ./

lint:
	go fix -diff ./...
	gofmt -d -l ./
	go vet ./...
	golangci-lint run

build:
	CGO_ENABLED=0 GOOS=linux go build -a ./cmd/totpcli/...

install:
	go install ./cmd/totpcli

all: fix tidy lint test
