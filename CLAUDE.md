# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this
repository.

## Structure

This is a Go workspace (`go.work`) with three modules:

- `shared/` — `github.com/majalcmaj/wind-alert/shared`: `model` (Location/Rule/Range types) and
  `dynamo` (DynamoDB access layer) used by both lambdas.
- `alert-job/` — `github.com/majalcmaj/wind-alert/alert-job`: the alerting Lambda. See
  `alert-job/CLAUDE.md` for its commands and architecture.
- `web/` — `github.com/majalcmaj/wind-alert/web`: the admin UI Lambda. See `web/ARCH.md` for its
  architecture.

`alert-job` and `web` each `replace github.com/majalcmaj/wind-alert/shared => ../shared` in
their `go.mod` so the workspace module resolves locally without a remote module.

## Commands

There is no `go.mod` at the repo root, so `./...` does not work from here — always pass explicit
module paths. The root `Makefile` wraps these:

```bash
make build   # go build ./shared/... ./alert-job/... ./web/... + both lambda binaries
make vet     # go vet   ./shared/... ./alert-job/... ./web/...
make test    # go test  ./shared/... ./alert-job/... ./web/...
make ci      # vet + test (what the pre-push hook and CI run)
make lint    # golangci-lint for alert-job and web
make fmt     # go fmt across all three modules
make clean   # clean alert-job and web build artifacts
```

The pre-push hook (`.husky/hooks/pre-push`) runs `make ci`.

Module-specific build/lint/Docker commands (`make build`, `make lint`, `make build-docker`, ...)
live in each module's own `Makefile` — run them from within `alert-job/` or `web/`.

### Local stack

```bash
make up    # build both lambda images, start shared DynamoDB Local + table setup, then both lambdas (detached)
make down  # tear everything down
make seed  # populate the shared local DynamoDB with sample data (scripts/seed-dynamodb.sh)
```

`make up` runs a single `dynamodb-local` (root `docker-compose.yml`, host port 8010 → container
8000) on the
`wind-alert-net` Docker network. `alert-job/docker-compose.yml` and `web/docker-compose.yml`
join that network as external, so both lambdas read/write the same
`wind-alert-locations`/`wind-alert-rules` tables, mirroring production. Running a module's
own `make run-local` / `make run-docker` standalone auto-creates the shared network and tables
via `ensure-dynamo`.

## Infrastructure & CI

- `terraform/` provisions both lambdas (ECR, IAM, Lambda function) and the shared DynamoDB
  tables (`wind-alert-locations`, `wind-alert-rules`) in one Terraform state.
- `.github/workflows/ci.yml` runs the full test suite on every push/PR, then deploys
  `alert-job` and/or `web` via the reusable `_lambda-pipeline.yml` workflow — only the lambda
  whose files changed (or both, if `shared/**`, `go.work`, or `go.work.sum` changed) gets
  redeployed, via `dorny/paths-filter`.
- `.github/workflows/terraform.yml` plans/applies `terraform/**` changes separately.

## Conventions

- Cross-package struct literals (e.g. `model.Range{...}`, `model.Rule{...}`) must use keyed
  fields — `go vet`'s composites check enforces this.
- Standardize on Go 1.26 across all three modules.
