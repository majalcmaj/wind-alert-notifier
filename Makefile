GO_PATHS = ./shared/... ./alert-job/... ./web/...

CHECK_DOCKER = @docker info >/dev/null 2>&1 || (echo "Error: Docker daemon not running" >&2; exit 1)

.PHONY: build vet test ci lint lint-fix fmt clean check-env build-docker up down up-dynamo down-dynamo seed

build:
	go build $(GO_PATHS)
	$(MAKE) -C alert-job build
	$(MAKE) -C web app

vet:
	go vet $(GO_PATHS)

test:
	go test $(GO_PATHS)

ci: vet test

lint:
	$(MAKE) -C alert-job lint
	$(MAKE) -C web lint

lint-fix:
	$(MAKE) -C alert-job lint-fix
	$(MAKE) -C web lint-fix

fmt:
	go fmt $(GO_PATHS)

clean:
	$(MAKE) -C alert-job clean
	$(MAKE) -C web clean

check-env:
	@test -f alert-job/.env || (echo "Missing alert-job/.env - copy from alert-job/.env.template" >&2; exit 1)
	@test -f web/.env || (echo "Missing web/.env - copy from web/.env.template" >&2; exit 1)

build-docker:
	$(CHECK_DOCKER)
	$(MAKE) -C alert-job build-docker
	$(MAKE) -C web build-docker

up-dynamo:
	$(CHECK_DOCKER)
	docker compose up -d dynamodb-local
	docker compose run --rm dynamo-setup

down-dynamo:
	docker compose down

up: check-env build-docker up-dynamo
	docker compose -f alert-job/docker-compose.yml up -d
	docker compose -f web/docker-compose.yml up -d

down:
	docker compose -f web/docker-compose.yml down
	docker compose -f alert-job/docker-compose.yml down
	$(MAKE) down-dynamo

seed:
	./scripts/seed-dynamodb.sh --endpoint-url http://localhost:8010
