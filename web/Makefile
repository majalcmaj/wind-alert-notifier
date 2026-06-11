docker-image=wind-alert:latest
GOLANGCI_LINT_PACKAGE ?= github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

GO_SRCS := $(shell find . -name '*.go' -not -path '*/vendor/*')
JS_SRCS := $(shell find . -name '*.js' -not -path '*/vendor/*' -not -path '*/node_modules/*')

bin/wind-alert-web: $(GO_SRCS) $(JS_SRCS)
	go build -tags lambda.norpc -o bin/wind-alert-web .

app: bin/wind-alert-web

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

bin/.docker-image-id: bin/wind-alert-web Dockerfile
	docker buildx build --platform linux/amd64 --provenance=false -t wind-alert:latest --iidfile $@ .

bin/rie-proxy: $(GO_SRCS)
	CGO_ENABLED=0 go build -o bin/rie-proxy ./cmd/rie-proxy

bin/.rie-proxy-image-id: bin/rie-proxy Dockerfile.rie-proxy
	docker buildx build --platform linux/amd64 --provenance=false -t wind-alert-rie-proxy:latest -f Dockerfile.rie-proxy --iidfile $@ .

.PHONY: build-docker
build-docker: bin/.docker-image-id bin/.rie-proxy-image-id

.PHONY: ensure-dynamo
ensure-dynamo:
	@docker network inspect wind-alert-net >/dev/null 2>&1 || $(MAKE) -C .. up-dynamo

.PHONY: run-docker
run-docker: build-docker ensure-dynamo
	docker compose up

run-test-request:
	@USER=$$(grep ^ADMIN_USER .env | cut -d= -f2); \
	 PASS=$$(grep ^ADMIN_PASSWORD .env | cut -d= -f2); \
	 AUTH=$$(printf '%s:%s' "$$USER" "$$PASS" | base64); \
	 curl -s "http://localhost:9080/2015-03-31/functions/function/invocations" \
	   -d "{\"version\":\"2.0\",\"routeKey\":\"GET /\",\"rawPath\":\"/\",\"headers\":{\"authorization\":\"Basic $$AUTH\"},\"requestContext\":{\"http\":{\"method\":\"GET\",\"path\":\"/\"}},\"isBase64Encoded\":false}"
