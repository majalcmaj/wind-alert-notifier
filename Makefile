GO_PATHS = ./shared/... ./alert-job/... ./web/...

CHECK_DOCKER = @docker info >/dev/null 2>&1 || (echo "Error: Docker daemon not running" >&2; exit 1)

.PHONY: build vet test ci lint lint-fix fmt clean

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

.PHONY: clean
clean:
	$(MAKE) -C alert-job clean
	$(MAKE) -C web clean

.PHONY: check-env
check-env:
	@test -f alert-job/.env || (echo "Missing alert-job/.env - copy from alert-job/.env.template" >&2; exit 1)
	@test -f web/.env || (echo "Missing web/.env - copy from web/.env.template" >&2; exit 1)

.PHONY: build
build:
	docker compose build

.PHONY: up-recreate
up-recreate: build
	docker compose up --force-recreate

.PHONY: up
up: check-env
	docker compose up 

