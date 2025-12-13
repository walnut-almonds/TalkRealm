MODULE_DIRS = .

SWAG ?= docker run --rm -v $(shell pwd):/code -w /code ghcr.io/swaggo/swag:v1.16.6

gowork:
	go work init .

tidy:
	go mod tidy

install-asdf:
	-asdf install

install: install-asdf tidy

fmt: install
	golangci-lint fmt -v ./...

fix: install
	golangci-lint run -v --fix ./...

lint: install
	golangci-lint run -v ./...

test: install
	go test -v -race -failfast ./...

.PHONY: docs
docs:
	$(SWAG) init -g ./internal/server/server.go -o ./docs/openapi --outputTypes json,yaml

check: fix lint test docs
