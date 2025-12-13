MODULE_DIRS = .

VERSION ?= $(shell git rev-parse --short HEAD)

.PHONY: gowork
gowork:
	go work init .

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: install-asdf
install-asdf:
	-asdf install

.PHONY: install
install: install-asdf tidy
	go install github.com/swaggo/swag/cmd/swag@v1.16.6

.PHONY: fmt
fmt: install
	golangci-lint fmt -v ./...

.PHONY: fix
fix: install
	golangci-lint run -v --fix ./...

.PHONY: lint
lint: install
	golangci-lint run -v ./...

.PHONY: test
test: install
	go test -v -race -failfast ./...

.PHONY: docs
docs: install
	swag init -g ./internal/server/server.go -o ./docs/openapi --outputTypes json,yaml

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build \
		-ldflags "-X github.com/walnut-almonds/talkrealm/buildinfo.Version=$(VERSION)" \
		-o ./bin/server ./cmd/server

.PHONY: check
check: fix lint test build docs
