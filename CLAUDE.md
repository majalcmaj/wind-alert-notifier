# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Structure

Single Go module (`module wind-alert`, Go 1.26). No `go.work`. `./...` works from root.

- `internal/model/` — `Location`, `Rule`, `Range` types (`dynamodbav`/`json` tags, `Rule.Describe()`); shared by both lambdas
- `internal/dynamo/` — `Store` (`dynamo.New(ctx)`), `LoadLocations`, `LoadRulesForLocation`
- `alert-job/` — alerting Lambda (see Alert-job lambda below)
- `web/` — admin UI Lambda (see Web lambda below + `web/ARCH.md`)
- `rie-proxy/` — local Lambda RIE proxy helper

## Commands

Root `Makefile` (builds `web` + `rie-proxy`, runs whole-module test/vet/lint):

```bash
make build         # go build web/main.go → bin/wind-alert-web (-tags lambda.norpc)
make build-rie-proxy
make test          # go test -v ./...
make test-coverage # tests + open HTML coverage report
make fmt           # go fmt ./...
make lint          # golangci-lint v2 (downloads if absent)
make lint-fix      # golangci-lint --fix
make build-docker  # linux/amd64 web image (build context = repo root)
make clean
```

Single test: `go test -v -run TestName ./alert-job/internal/`

Subdirs `alert-job/` and `web/` have their own `Makefile` + `docker-compose.yml`:
- `alert-job/`: `make build` (→ `bin/alert-job`), `build-docker`, `run-test-request`, `invoke-lambda`, `generate` (compile `mail_template.mjml` → `.html`, needs `npx`)
- `web/`: `make build`, `lint`, `build-docker`

Pre-push hook runs the test suite — failing tests block push.

## Alert-job lambda

AWS Lambda (Docker/ECR) that fetches wind forecasts, evaluates rules, emails matches via SES.

**Flow (`alert-job/main.go`):**
1. Constructs forecast providers (OpenWeather, ICM-Meteo, Open-Meteo, YR.no); tokens from env (`OPENWEATHER_TOKEN`, `ICM_METEO_TOKEN`, …)
2. `aggregator.go` combines providers into `WeatherReading` slices
3. `rule_engine.go` evaluates each `WindDataPoint` against `[]model.Rule` (`RunRuleEngine`, `EvaluateForecast`, `EvaluateWithConfidence`); cyclic range checks live on `model.Range`
4. `mail_renderer.go` renders `mail_template.html` (`//go:embed`, `windArrow` func); `wind_arrow_renderer.go` maps degree → Unicode arrow
5. Sends via SES v2, or prints to stdout when `LOCAL_MODE=true`

**Mail template:** edit `alert-job/internal/mail_template.mjml`, regenerate `.html` with `make generate`. `.html` is committed + embedded at compile time.

Local run: copy `.env.template` → `.env`, set `OPENWEATHER_TOKEN`.

## Web lambda

Single lambdalith behind Lambda Function URL. Only writer to shared DynamoDB; `alert-job` read-only.

- **Routing:** Go 1.22 `net/http.ServeMux` → `httpadapter` (API-GW payload 2.0). Plain `net/http` handlers, testable with `httptest`.
- **Frontend:** Server-rendered HTML + htmx fragments. Templates + static (htmx, Pico.css) via `//go:embed`. No CDN, no build step.
- **Auth:** Basic-auth middleware; `ADMIN_USER`/`ADMIN_PASSWORD` env vars. Function URL `authorization_type = "NONE"`.
- **Packages:** `web/internal/server/` (router/handlers/middleware), `web/internal/validate/` (Location/Rule form validation), `web/internal/web/` (templates + static).
- **Rule rename:** changing `name` rewrites DynamoDB SK — delete-old + put-new.
- **Cyclic ranges:** `angle.from > angle.to` and `hour.from > hour.to` are valid (wrap); don't reject.

See `web/ARCH.md` for route table + data model.

## Infrastructure & CI

- `terraform/` — ECR, IAM, Lambda, DynamoDB tables in one state
- `.github/workflows/ci.yml` — test + deploy; only changed lambda redeploys (`dorny/paths-filter`); `internal/**` changes trigger both
- `.github/workflows/terraform.yml` — separate plan/apply for `terraform/**`

## Conventions

- Cross-package struct literals must use keyed fields (`go vet` composites check enforces this)
- Go 1.26
