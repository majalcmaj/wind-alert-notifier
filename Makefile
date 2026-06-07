docker-image = wind-alert:latest
GOLANGCI_LINT_PACKAGE ?= github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
GO_SOURCES = $(shell find . -name '*.go' -not -path './vendor/*')

CHECK_DOCKER = @docker info >/dev/null 2>&1 || (echo "Error: Docker daemon not running" >&2; exit 1)

.PHONY: clean test test-coverage fmt lint lint-fix build build-docker run-docker run-local stop-local run-test-request invoke-lambda generate

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

run-local: .docker-built
	$(CHECK_DOCKER)
	docker compose up

stop-local:
	docker compose down

run-test-request:
	curl "http://localhost:9000/2015-03-31/functions/function/invocations" -d '{}'

invoke-lambda:
	aws lambda invoke --region eu-central-1 --function-name wind-alert --cli-binary-format raw-in-base64-out --payload '{}' /tmp/wind-alert-invoke.json
	@cat /tmp/wind-alert-invoke.json | jq .
	@aws logs tail /aws/lambda/wind-alert --region eu-central-1 --since 2m

internal/mail_template.html: internal/mail_template.mjml
	npx -y mjml $< -o $@

generate: internal/mail_template.html
