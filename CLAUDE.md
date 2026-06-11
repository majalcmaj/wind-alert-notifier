# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this
repository.

## Structure

This is a Go workspace (`go.work`) with three modules:

- `shared/` — `github.com/majalcmaj/wind-alert/shared`: `model` (Location/Rule/Range types) and
  `dynamo` (DynamoDB access layer) used by both lambdas.
- `forecaster/` — `github.com/majalcmaj/wind-alert/forecaster`: the alerting Lambda. See
  `forecaster/CLAUDE.md` for its commands and architecture.
- `web/` — `github.com/majalcmaj/wind-alert/web`: the admin UI Lambda. See `web/ARCH.md` for its
  architecture.

`forecaster` and `web` each `replace github.com/majalcmaj/wind-alert/shared => ../shared` in
their `go.mod` so the workspace module resolves locally without a remote module.

## Commands

There is no `go.mod` at the repo root, so `./...` does not work from here — always pass explicit
module paths:

```bash
go build ./shared/... ./forecaster/... ./web/...
go vet   ./shared/... ./forecaster/... ./web/...
go test  ./shared/... ./forecaster/... ./web/...
```

The pre-push hook (`.husky/hooks/pre-push`) runs `go vet` and `go test` with these patterns.

Module-specific build/lint/Docker commands (`make build`, `make lint`, `make build-docker`, ...)
live in each module's own `Makefile` — run them from within `forecaster/` or `web/`.

## Infrastructure & CI

- `terraform/` provisions both lambdas (ECR, IAM, Lambda function) and the shared DynamoDB
  tables (`wind-alert-locations`, `wind-alert-rules`) in one Terraform state.
- `.github/workflows/ci.yml` runs the full test suite on every push/PR, then deploys
  `forecaster` and/or `web` via the reusable `_lambda-pipeline.yml` workflow — only the lambda
  whose files changed (or both, if `shared/**`, `go.work`, or `go.work.sum` changed) gets
  redeployed, via `dorny/paths-filter`.
- `.github/workflows/terraform.yml` plans/applies `terraform/**` changes separately.

## Conventions

- Cross-package struct literals (e.g. `model.Range{...}`, `model.Rule{...}`) must use keyed
  fields — `go vet`'s composites check enforces this.
- Standardize on Go 1.26 across all three modules.
