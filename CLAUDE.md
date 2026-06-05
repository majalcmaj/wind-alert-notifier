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
make run-docker        # run locally via Lambda RIE on :9000
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
- `rule_engine.go` — evaluates a `WindDataPoint` against `[]Rule`; each rule has `AngleRange`, `SpeedRange`, `HourRange`; angle/hour use cyclic range logic (`From > To` wraps around)
- `mail_renderer.go` — executes `mail_template.html` (embedded via `//go:embed`) with a `windArrow` template func
- `wind_arrow_renderer.go` — maps wind degree → Unicode directional arrow (8 directions, text variation selector `︎`)

**Email template:**
- Source: `internal/mail_template.mjml` — edit this, not the HTML
- Generated: `internal/mail_template.html` — committed output, regenerate with `make generate`
- Embedded at compile time; changing `.html` takes effect without any registration step

**Deployment:**
- `aws/template.yml` — SAM template; Lambda pulls image from ECR repository `wind-alert-docker-repository`
- `terraform/` — infrastructure provisioning
- `.github/workflows/deploy-docker.yml` — CI/CD pipeline

## Environment

Copy `.env.template` → `.env` and set `OPENWEATHER_TOKEN`. The `.env` file is used by `make run-docker`.
