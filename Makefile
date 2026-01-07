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
	./scripts/setup_asdf.sh

.PHONY: install-kubectl
install-kubectl: install-asdf
	./scripts/setup_kubectl.sh

.PHONY: reinstall-kubectl
reinstall-kubectl:
	rm -rf ~/.kube
	$(MAKE) --no-print-directory install-kubectl

.PHONY: install
install: install-asdf install-kubectl
	go mod download

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
build: install
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build \
		-ldflags "-X github.com/walnut-almonds/talkrealm/buildinfo.Version=$(VERSION)" \
		-o ./bin/server ./cmd/server

pack: build
	docker buildx build \
		--platform linux/amd64 \
		--load \
		-t talk-realm:$(VERSION) \
		-t talk-realm:latest \
		--build-arg APP=bin/server \
		.

.PHONY: k8s-local
k8s-local: install
	mkdir -p ./build
	kubectl kustomize ./deploy/k8s/overlays/local > ./build/local.yaml

.PHONY: k8s-dev
k8s-dev: install
	mkdir -p ./build
	kubectl kustomize ./deploy/k8s/overlays/dev > ./build/dev.yaml

.PHONY: k8s-local-delete
k8s-local-delete:
	kubectl delete namespace talk-realm-local

.PHONY: k8s-dev-delete
k8s-dev-delete:
	kubectl delete namespace talk-realm-dev

.PHONY: k8s-local-deploy
k8s-local-deploy: pack k8s-local
	kubectl apply --prune -l app.kubernetes.io/namespace=talk-realm-local,app.kubernetes.io/name=talkrealm,prunable=true -f ./build/local.yaml
	kubectl rollout restart deployment/talk-realm -n talk-realm-local

.PHONY: k8s-dev-deploy
k8s-dev-deploy: pack k8s-dev
	kubectl apply --prune -l app.kubernetes.io/namespace=talk-realm-dev,app.kubernetes.io/name=talkrealm,prunable=true -f ./build/dev.yaml
	kubectl rollout restart deployment/talk-realm -n talk-realm-dev

.PHONY: check
check: tidy fix lint test build docs

.PHONY: port-forward-local
port-forward-local:
	kubectl port-forward svc/talk-realm 8080:8080 -n talk-realm-local

.PHONY: port-forward-dev
port-forward-dev:
	kubectl port-forward svc/talk-realm 8080:8080 -n talk-realm-dev

.PHONY: port-forward-redis-local
port-forward-redis-local:
	kubectl port-forward svc/redis 6379:6379 -n talk-realm-local

.PHONY: port-forward-postgres-local
port-forward-postgres-local:
	kubectl port-forward svc/postgres 5432:5432 -n talk-realm-local
