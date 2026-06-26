# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Structure

Go workspace (`go.work`), three modules:

- `shared/` — `model` (Location/Rule/Range) + `dynamo` (DynamoDB Store); used by both lambdas
- `alert-job/` — alerting Lambda; see `alert-job/CLAUDE.md`
- `web/` — admin UI Lambda; see `web/ARCH.md`

Both lambdas `replace github.com/majalcmaj/wind-alert/shared => ../shared` in `go.mod`.

## Commands

No `go.mod` at root — `./...` doesn't work. Use root `Makefile`:

```bash
make build   # go build ./shared/... ./alert-job/... ./web/... + lambda binaries
make vet     # go vet across all modules
make test    # go test across all modules
make ci      # vet + test (pre-push hook + CI)
make lint    # golangci-lint for alert-job and web
make fmt     # go fmt across all modules
make clean   # clean build artifacts
```

Single test (from module dir): `go test -v -run TestName ./internal/`

Module-specific `make build/lint/build-docker`: run from `alert-job/` or `web/`.
`alert-job/` has `make run-local`/`make run-docker` (auto-creates network + tables via `ensure-dynamo`). `web/` does not.

### Local stack

```bash
make up    # build images, start DynamoDB Local + table setup, start both lambdas (detached)
make down  # tear down
make seed  # populate DynamoDB Local with sample data (scripts/seed-dynamodb.sh)
```

`make up` runs single `dynamodb-local` (host port 8010→8000) on `wind-alert-net`. Both module compose files join as external — both lambdas share `wind-alert-locations`/`wind-alert-rules`.

## Web lambda

Single lambdalith behind Lambda Function URL. Only writer to shared DynamoDB; `alert-job` read-only.

- **Routing:** Go 1.22 `net/http.ServeMux` → `httpadapter` (API-GW payload 2.0). Plain `net/http` handlers, testable with `httptest`.
- **Frontend:** Server-rendered HTML + htmx fragments. Templates + static (htmx, Pico.css) via `//go:embed`. No CDN, no build step.
- **Auth:** Basic-auth middleware; `ADMIN_USER`/`ADMIN_PASSWORD` env vars. Function URL `authorization_type = "NONE"`.
- **Packages:** `internal/server/` (router/handlers/middleware), `internal/validate/` (Location/Rule form validation), `internal/web/` (templates + static).
- **Rule rename:** changing `name` rewrites DynamoDB SK — delete-old + put-new.
- **Cyclic ranges:** `angle.from > angle.to` and `hour.from > hour.to` are valid (wrap); don't reject.

See `web/ARCH.md` for route table + data model.

## Infrastructure & CI

- `terraform/` — ECR, IAM, Lambda, DynamoDB tables in one state
- `.github/workflows/ci.yml` — test + deploy; only changed lambda redeploys (`dorny/paths-filter`); shared/** changes trigger both
- `.github/workflows/terraform.yml` — separate plan/apply for `terraform/**`

## Conventions

- Cross-package struct literals must use keyed fields (`go vet` composites check enforces this)
- Go 1.26 across all three modules
