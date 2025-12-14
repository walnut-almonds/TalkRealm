MODULE_DIRS = .

VERSION ?= g$(shell git rev-parse --short HEAD)

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

.PHONY: check
check: fix lint test

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build \
		-ldflags "-X github.com/walnut-almonds/talkrealm/buildinfo.Version=$(VERSION)" \
		-o ./bin/server ./cmd/server
