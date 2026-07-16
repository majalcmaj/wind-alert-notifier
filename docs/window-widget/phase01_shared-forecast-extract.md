<!-- plan-status: pending -->
# Phase 01 — shared-forecast-extract

> **Status:** ⬜ PENDING

Read `docs/window-widget/prompt.md` first.

## Goal
Provider clients and forecast types live in a new `shared/forecast` package importable
by both lambdas; `alert-job` behavior is byte-identical. This unblocks the web lambda
fetching forecasts (phase 04) without duplicating provider code.

## Red
Create `shared/forecast/forecast_test.go` asserting the target public API exists:

```go
package forecast_test

import (
	"testing"

	"github.com/majalcmaj/wind-alert/shared/forecast"
)

func TestPublicAPIExists(t *testing.T) {
	var _ forecast.Provider = forecast.NewYrNo("https://api.met.no")
	var _ func(name string) (forecast.Provider, error)
	var _ forecast.WindDataPoint
	var _ forecast.ProviderReading
}
```

Run `make test` — fails to compile: package `shared/forecast` does not exist. That is
the red.

## Green
Mechanical move, no behavior change:

1. `git mv` from `alert-job/internal/` to `shared/forecast/`:
   `openweather.go`, `yrno.go`, `openmeteo.go`, `icmmeteo.go`, `aggregator.go`
   and their `_test.go` files (test fixtures included, if any are file-based).
2. Change `package internal` → `package forecast` in moved files.
3. Types that move with them: `WindDataPoint`, `WeatherReading`, `ProviderReading`,
   `NamedForecaster`, `Forecaster`, `FetchAll`.
4. What does **not** move (evaluation/presentation, handled in phase 02):
   `rule_engine.go`, `mail_renderer.go`, `wind_arrow_renderer.go`, `mail_template.*`.
   The `Matched bool` field on `WindDataPoint` moves along for now — phase 02 removes it.
5. Update imports in `alert-job/internal/*` and `alert-job/main.go` to
   `github.com/majalcmaj/wind-alert/shared/forecast`. `go.work` already spans the
   modules; `alert-job/go.mod` already `replace`s `../shared`, so no dependency edits.

Run `make test` — the red test compiles and passes, all existing alert-job tests pass.

## Refactor
- Rename for clarity now that the API is public: `Forecaster` → `forecast.Provider`,
  `NamedForecaster` → `forecast.NamedProvider`, `FetchAll` → `forecast.FetchAll`
  (already fine). One provider = one file = one constructor `NewX(baseURL, …)`.
- Give every provider constructor the same shape and inject the `*http.Client`
  (default when nil) so tests stop spinning `httptest` servers for timeout config:
  `NewOpenWeather(baseURL, token string, opts ...Option)` is over-engineering — a
  plain optional client parameter or a package-level default is enough. Pick the
  simplest uniform signature and apply it to all four.
- Delete any now-unused identifiers left behind in `alert-job/internal`.
- Cross-package struct literals now cross a module boundary: keyed fields everywhere
  (`go vet` composites check enforces this — see root CLAUDE.md).

## Verify
- `make ci` (vet + test across all three modules) green.
- `make build` green (lambda binaries still compile).
- `cd alert-job && make run-local` + `make run-test-request`: response identical in
  shape to before the move (spot-check one provider present in output).

## Commit
`refactor(shared): extract forecast providers into shared/forecast`
