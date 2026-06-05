docker-image = wind-alert:latest
GOLANGCI_LINT_PACKAGE ?= github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
GO_SOURCES = $(shell find . -name '*.go' -not -path './vendor/*')

CHECK_DOCKER = @docker info >/dev/null 2>&1 || (echo "Error: Docker daemon not running" >&2; exit 1)

.PHONY: clean test test-coverage fmt lint lint-fix build build-docker run-docker run-test-request generate

clean:
	rm -rf bin test-coverage.out .docker-built

test:
	go test -v ./...

test-coverage:
	go test -timeout=30s -cover -coverprofile test-coverage.out ./... && go tool cover -html=test-coverage.out

bin/wind-alert-go: $(GO_SOURCES) go.mod go.sum
	go build -tags lambda.norpc -o bin/ ./...

build: bin/wind-alert-go

fmt:
	go fmt ./...

lint:
	go run $(GOLANGCI_LINT_PACKAGE) run

lint-fix:
	go run $(GOLANGCI_LINT_PACKAGE) run --fix

.docker-built: Dockerfile $(GO_SOURCES) go.mod go.sum
	$(CHECK_DOCKER)
	docker buildx build --platform linux/amd64 --provenance=false -t $(docker-image) .
	@touch $@

build-docker: .docker-built

run-docker: .docker-built
	$(CHECK_DOCKER)
	docker run --name wind-alert --rm -p 9000:8080 --env-file .env --entrypoint /usr/local/bin/aws-lambda-rie $(docker-image) ./main

run-test-request:
	curl "http://localhost:9000/2015-03-31/functions/function/invocations" -d '{}'

internal/mail_template.html: internal/mail_template.mjml
	npx -y mjml $< -o $@

generate: internal/mail_template.html
