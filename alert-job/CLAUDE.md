# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make test              # run all tests
make test-coverage     # tests + open HTML coverage report
make build             # compile with -tags lambda.norpc → bin/
make lint              # golangci-lint v2 (downloads if absent)
make lint-fix          # golangci-lint --fix
make generate          # compile mail_template.mjml → mail_template.html (requires npx)
make build-docker      # build linux/amd64 Docker image
make run-local         # run locally via Lambda RIE on :9000 (auto-starts shared DynamoDB Local)
make run-test-request  # POST {} to local Lambda RIE
```

Single test: `go test -v -run TestName ./internal/`

Pre-push hook runs the test suite — failing tests block push.

## Architecture

AWS Lambda function (Docker/ECR deployment) that fetches wind forecasts and emails them via SES.

**Flow:**
1. `main.go` handler fetches forecast from OpenWeather One Call 3.0 API for hardcoded coordinates (54.646034, 18.512407 — Gdańsk/Sopot, Poland)
2. Renders HTML mail with `internal.RenderMail`
3. Sends via AWS SES v2 (hardcoded sender/recipient in `main.go`)

**`internal/` packages:**
- `openweather.go` — HTTP client for OpenWeather; maps response into `WeatherReading` (hourly + daily `WindDataPoint` slices)
- `rule_engine.go` — evaluates a `WindDataPoint` against `[]model.Rule` (matching, `RunRuleEngine`, `EvaluateForecast`, `EvaluateWithConfidence`); cyclic range checks (`From > To` wraps around) live on `model.Range` itself
- `mail_renderer.go` — executes `mail_template.html` (embedded via `//go:embed`) with a `windArrow` template func
- `wind_arrow_renderer.go` — maps wind degree → Unicode directional arrow (8 directions, text variation selector `︎`)

**Shared module (`../shared`):**
- `model` — `Location`, `Rule`, `Range` types (with `dynamodbav`/`json` tags) and `Rule.Describe()`, shared with `../web`
- `dynamo` — `Store` (`dynamo.New(ctx)`), `LoadLocations`, `LoadRulesForLocation` used by `main.go`

**Email template:**
- Source: `internal/mail_template.mjml` — edit this, not the HTML
- Generated: `internal/mail_template.html` — committed output, regenerate with `make generate`
- Embedded at compile time; changing `.html` takes effect without any registration step

**Docker build:**
- Build context is the **monorepo root** (`..` from this dir), so `go.work` and `../shared` are visible to the build — see `Dockerfile` and the `.docker-built` target in `Makefile`
- `aws/template.yml` — legacy SAM template, unused (kept for reference only)
- `../terraform/` — infrastructure provisioning (shared with `../web`)
- `.github/workflows/` — CI/CD pipeline

## Environment

Copy `.env.template` → `.env` and set `OPENWEATHER_TOKEN`. The `.env` file is used by `make run-local`.
