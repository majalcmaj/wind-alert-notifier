<!-- plan-status: pending -->
# Phase 04 — forecast-endpoint

> **Status:** ⬜ PENDING

Read `docs/window-widget/prompt.md` first.

## Goal
The web lambda serves `GET /locations/{id}/forecast` (HTML page shell) and
`GET /locations/{id}/forecast/data` (JSON payload). The payload is fully pre-chewed:
knots, aggregates, and per-rule 3-state hourly matches computed in Go. The TypeScript
widget (phases 05–07) will be a pure renderer of this payload.

## Red
`web/internal/server/forecast_test.go` (fails: 404, types don't exist):

```go
func TestForecastData(t *testing.T) {
	srv := New(fakeStore, fakeFetcher) // fetcher returns canned readings for 2 providers
	res := httptest GET "/locations/sopot/forecast/data"
	// assert 200, content-type application/json
	// decode into server.ForecastView and assert:
	//  - len(Hours) == len(Aggregate.MedianKn) == len(each provider series)
	//  - Providers sorted by name; failed provider absent from Providers, present in Issues
	//  - Rules[0].HourlyState values ∈ {0,1,2} and match rules.Evaluate recomputed here
	//  - speeds are knots (fixture is m/s; assert ×1.9438445)
}
```

Also a golden-payload test: `TestForecastViewGolden` marshals a `ForecastView` built
from fixed inputs and compares byte-identical against
`web/frontend/fixtures/forecast-payload.json` (create the dir now; the TS side parses
the same file in phase 05 — this is the cross-language contract).

## Green
1. **Seam** — extend the server constructor, mirroring the existing `Datastore`
   pattern:

```go
type ForecastFetcher interface {
	Fetch(ctx context.Context, loc model.Location) []forecast.ProviderReading
}
```

Production implementation wraps `forecast.FetchAll` with the four providers
constructed in `web/main.go` (tokens from env: `OPENWEATHER_TOKEN`,
`ICM_METEO_TOKEN` — same names as alert-job). Tests inject a fake.

2. **View assembly** — a pure function, no HTTP, no AWS; this is where all the domain
   thinking lives and it must be unit-testable in isolation:

```go
// BuildForecastView merges provider readings onto a single hourly axis,
// converts to knots, aggregates, and evaluates every rule per hour.
func BuildForecastView(loc model.Location, rls []model.Rule,
	readings []forecast.ProviderReading, now time.Time) ForecastView
```

Payload shape (mirror of the reference design's needs — keep field names exactly, the
TS types copy them):

```json
{
  "location": {"id": "sopot", "name": "Sopot", "lat": 54.646, "lon": 18.512, "timezone": "Europe/Warsaw"},
  "generatedAt": "2026-07-12T10:00:00Z",
  "hours": ["2026-07-12T10:00:00Z", "2026-07-12T11:00:00Z"],
  "providers": [
    {"name": "openweather", "speedKn": [14.2, 15.1], "gustKn": [19.0, 21.3], "dirDeg": [310, 315]}
  ],
  "aggregate": {"medianKn": [14.0, 15.0], "minKn": [12.1, 13.4], "maxKn": [15.8, 16.9], "dirDeg": [312, 316]},
  "rules": [
    {"name": "Solid NW", "speedKn": [12, 25], "dirDeg": [270, 360], "hours": [6, 20], "hourlyState": [2, 1]}
  ],
  "issues": [{"provider": "icm-meteo", "error": "..."}]
}
```

Ideas the executor must not miss:

- **Hour axis**: providers return different lengths/steps. The axis is the sorted
  union of hourly timestamps within `[now, now+48h]`; a provider missing an hour gets
  `null` in its series (JSON `null`, Go `[]*float64` or `math.NaN`-encoded — pick
  `*float64`, explicit beats sentinel).
- **Aggregate direction is a circular mean**, not an arithmetic one — the average of
  350° and 10° is 0°, not 180°:

```go
func circularMeanDeg(degs []float64) float64 {
	var x, y float64
	for _, d := range degs {
		r := d * math.Pi / 180
		x, y = x+math.Cos(r), y+math.Sin(r)
	}
	return math.Mod(math.Atan2(y, x)*180/math.Pi+360, 360)
}
```

- **Median of an even count** = mean of the middle two; aggregate over only the
  providers that have data at that hour.
- **`hourlyState`** = `int(rules.Evaluate(rule, dp, loc.TZ()))` on the *aggregate*
  series — the lanes reflect the consensus wind, matching what the alert engine would
  broadly do. Provider-level lanes are out of scope.
- Unit conversion happens exactly once, here: `kn = ms * 1.9438445`. Grep-able
  constant `const metersPerSecondToKnots = 1.9438445` in one place.

3. **Handler + cache**: handlers stay thin (parse path, call service, encode).
   In-memory TTL cache (10 min) keyed by location id around the fetch — a
   `map[string]entry` + `sync.Mutex`, ~30 lines, no dependency. Lambda warm
   containers benefit; cold starts just fetch.

4. **Page shell**: `GET /locations/{id}/forecast` renders a template with the
   widget mount `<div id="window-widget" data-payload-url="...">` and a script tag
   for the (phase 05) bundle. Add a "Forecast" link per location row. Basic-auth
   middleware already wraps all routes — nothing to do.

5. **Terraform** (`terraform/web.tf`): add the two token env vars (sensitive
   variables, same pattern as alert-job's).

## Refactor
- If `BuildForecastView` exceeds ~80 lines, split by named intent:
  `mergeHourAxis`, `resampleProvider`, `aggregate`, `evaluateRules` — not by line
  count. Each helper takes and returns plain data.
- Delete any leftover placeholder JSON encoding; one `respondJSON(w, v)` helper next
  to the existing `respond.go` HTML helpers.

## Verify
- `make ci` green; golden fixture committed and matched.
- `make up && make seed`, then
  `curl -u admin:admin localhost:PORT/locations/sopot/forecast/data | jq` — real
  provider data (needs tokens in `.env`), knots plausible, `hourlyState` present.
- Kill one token → provider appears under `issues`, page still 200.

## Commit
`feat(web): forecast page shell + aggregated forecast JSON endpoint`
