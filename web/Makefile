docker-image ?= wind-alert-web:latest
GOLANGCI_LINT_PACKAGE ?= github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

CHECK_DOCKER = @docker info >/dev/null 2>&1 || (echo "Error: Docker daemon not running" >&2; exit 1)

GO_SRCS := $(shell find . -name '*.go' -not -path '*/vendor/*')
JS_SRCS := $(shell find . -name '*.js' -not -path '*/vendor/*' -not -path '*/node_modules/*')
BIN_DIR ?= bin

.PHONY: build build-docker build-rie-proxy clean test test-coverage fmt lint lint-fix

$(BIN_DIR)/wind-alert-web: $(GO_SRCS) $(JS_SRCS)
	go build -tags lambda.norpc -o $(BIN_DIR)/wind-alert-web web/main.go

build: $(BIN_DIR)/wind-alert-web

clean:
	[ ! -d bin ] || rm -rf bin

test:
	go test -v ./...

test-coverage:
	go test -timeout=30s -cover -coverprofile test-coverage.out ./... && go tool cover -html=test-coverage.out

fmt:
	go fmt ./...

lint:
	go run $(GOLANGCI_LINT_PACKAGE) run

lint-fix:
	go run $(GOLANGCI_LINT_PACKAGE) run --fix

bin/.docker-image-id: $(BIN_DIR)/wind-alert-web Dockerfile
	$(CHECK_DOCKER)
	docker buildx build --platform linux/amd64 --provenance=false -t $(docker-image) --iidfile $@ -f Dockerfile ..

build-docker: bin/.docker-image-id

$(BIN_DIR)/rie-proxy: $(GO_SRCS)
	CGO_ENABLED=0 go build -o $(BIN_DIR)/rie-proxy ./rie-proxy
build-rie-proxy: $(BIN_DIR)/rie-proxy

bin/.rie-proxy-image-id: $(BIN_DIR)/rie-proxy Dockerfile.rie-proxy
	docker buildx build --platform linux/amd64 --provenance=false -t wind-alert-rie-proxy:latest -f Dockerfile.rie-proxy --iidfile $@ ..
