MODULE_DIRS = .

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

check: fix lint test
