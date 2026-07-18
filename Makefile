docker-image ?= wind-alert-web:latest
GOLANGCI_LINT_PACKAGE ?= github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

CHECK_DOCKER = @docker info >/dev/null 2>&1 || (echo "Error: Docker daemon not running" >&2; exit 1)
DOCKER_BUILD = @docker buildx build --platform linux/amd64 --provenance=false -t $(docker-image) --iidfile $@ .
DOCKER_BUILD_LAMBDA = $(DOCKER_BUILD) --file docker/lambda.Dockerfile

INTERNAL_SRCS := $(shell find internal/ -type f -not -path '*/vendor/*')
WEB_SRCS := $(shell find web/ -type f -not -path '*/vendor/*') $(INTERNAL_SRCS)
JOB_SRCS := $(shell find alert-job/ -type f -not -path '*/vendor/*') $(INTERNAL_SRCS)
BIN_DIR ?= bin

# =============== BUILD ===============

.PHONY: build
build: build-web build-job build-rie-proxy

$(BIN_DIR)/wind-alert-job/bootstrap: $(JOB_SRCS)
	go build -tags lambda.norpc -o $(BIN_DIR)/wind-alert-job/bootstrap ./alert-job
.PHONY: build-job
build-job: $(BIN_DIR)/wind-alert-job/bootstrap

$(BIN_DIR)/wind-alert-web/bootstrap: $(WEB_SRCS) $(JS_SRCS)
	go build -tags lambda.norpc -o $(BIN_DIR)/wind-alert-web/bootstrap ./web
.PHONY: build-web
build-web: $(BIN_DIR)/wind-alert-web/bootstrap

$(BIN_DIR)/rie-proxy: docker/rie-proxy/main.go
	CGO_ENABLED=0 go build -o $(BIN_DIR)/rie-proxy ./docker/rie-proxy
.PHONY: build-rie-proxy
build-rie-proxy: $(BIN_DIR)/rie-proxy

alert-job/internal/mail_template.html: alert-job/internal/mail_template.mjml
	npx -y mjml $< -o $@

.PHONY: render-mail-template
render-mail-template: alert-job/internal/mail_template.html

.PHONY: clean
clean:
	[ ! -d bin ] || rm -rf bin

# =============== DOCKER ===============

bin/.web-docker-image-id: $(BIN_DIR)/wind-alert-web docker/lambda.Dockerfile
	$(CHECK_DOCKER)
	$(DOCKER_BUILD_LAMBDA) --build-arg binary="bin/wind-alert-web"
.PHONY: build-web-docker
build-web-docker: bin/.web-docker-image-id

bin/.job-docker-image-id: $(BIN_DIR)/wind-alert-job docker/lambda.Dockerfile
	$(CHECK_DOCKER)
	$(DOCKER_BUILD_LAMBDA) --build-arg binary="bin/wind-alert-job"
.PHONY: build-job-docker
build-job-docker: bin/.job-docker-image-id

bin/.rie-proxy-image-id: $(BIN_DIR)/rie-proxy docker/rie-proxy.Dockerfile
	$(CHECK_DOCKER)
	$(DOCKER_BUILD) --file docker/rie-proxy.Dockerfile
.PHONY: build-rie-proxy-docker
build-rie-proxy-docker: bin/.rie-proxy-image-id

.PHONY: up
up: build
	$(CHECK_DOCKER)
	@docker compose -f docker/docker-compose.yml up

# =============== TESTS/CHECKS ===============

.PHONY: test
test:
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	go test -timeout=30s -cover -coverprofile test-coverage.out ./... && go tool cover -html=test-coverage.out

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	go run $(GOLANGCI_LINT_PACKAGE) run

.PHONY: lint-fix
lint-fix:
	go run $(GOLANGCI_LINT_PACKAGE) run --fix

.PHONY: vet
vet:
	go vet ./...

.PHONY: ci
ci: lint vet test
# =============== JOB LAMBDA ===============

.PHONY: run-job
run-job:
	curl "http://localhost:9090/2015-03-31/functions/function/invocations" -d '{}'

.PHONY: run-job-aws
run-job-aws:
	aws lambda invoke --region eu-central-1 --function-name wind-alert-job --cli-binary-format raw-in-base64-out --payload '{}' /tmp/wind-alert-invoke.json
	@cat /tmp/wind-alert-invoke.json | jq .
	@aws logs tail /aws/lambda/wind-alert --region eu-central-1 --since 2m
