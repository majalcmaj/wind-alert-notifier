# Add ICM Meteo (api.meteo.pl) as a new forecast provider

## Context

Wind-alert aggregates wind forecasts from several providers (`openweather`, `yrno`, `openmeteo`) through the `Forecaster` interface and computes a confidence score across them (`internal.EvaluateWithConfidence`). The user wants to add ICM (Interdisciplinary Centre for Mathematical and Computational Modelling, University of Warsaw) `api.meteo.pl` as a fourth provider to widen the pool of independent forecasts and improve confidence accuracy. User confirmed: they have a **paid ICM token** (forecast endpoint costs money — 402 if unfunded), and want to use the **coamps model, height-surface (zht) wind fields**.

ICM's API is structurally very different from the existing three: it has no precomputed wind-speed/direction field — only raw U/V wind vector components — and forecast retrieval requires resolving lat/lon to a model grid cell first, then picking the latest available model run.

## ICM API shape (from docs/meteo + live discovery needed)

- Base: `https://api.meteo.pl`, header `Authorization: Token {token}` on every call
- Hierarchical path: `/api/v1/model/{model}/grid/{grid}/coordinates/{row},{col}/field/{field}/level/{level}/date/{date}/forecast/`
- `coordinates` are **grid row/col ints**, not lat/lon — convert via `GET /api/v1/model/{model}/grid/{grid}/latlon2rowcol/{lat},{lon}/` → `{"points":[{"row":R,"col":C}]}` (free, not flagged paid)
- `date` = a **model run** timestamp `yyyy-mm-ddTHH` (HH ∈ 00/06/12/18/24); list available runs via `GET .../date/` → run-length-encoded `{"dates":[{"starting-date":D,"count":N,"interval":H}, ...]}`. Latest run = last entry's `starting-date + (count-1)*interval` hours.
- `POST .../forecast/` → `{"times":[...], "data":[...]}` parallel arrays, **paid** (402 if account unfunded)
- Wind fields: no combined speed/dir; use `uuwind_zht_fcstfld` (U / x-component) and `vvwind_zht_fcstfld` (V / y-component) at "height surface" level — per user's choice. **Open item for implementer**: query the live API once (with the token) to pin down the actual `grid` name (must cover 54.646034, 18.512407) and the actual `level` value both fields expose (e.g. "10" for 10 m) — hardcode these as constants the same way the project already hardcodes the Gdańsk/Sopot coordinates.

### Wind vector → speed/angle

Meteorological convention (`WindAngle` = direction wind blows **from**, matching `wind_deg`/`wind_from_direction` used by the other providers):

```go
speed := math.Hypot(u, v)
angle := math.Mod(math.Atan2(-u, -v)*180/math.Pi+360, 360)
```

## Implementation

### 1. New file `internal/icmmeteo.go`

Mirror `internal/openweather.go` / `internal/openmeteo.go` patterns (constructor validation, `errors.Wrap` with `"icm:"` prefix, `http.DefaultClient`, deferred body-close with stderr warning, `//go:embed`-free plain JSON parsing).

```go
type IcmMeteo struct {
    baseURL string
    token   string
}

func NewIcmMeteo(baseURL, token string) (*IcmMeteo, error) // validates non-empty, like NewOpenWeather

const (
    icmModel = "coamps"
    icmGrid  = "<discovered-grid-name>"
    icmLevel = "<discovered-level>"
    icmFieldU = "uuwind_zht_fcstfld"
    icmFieldV = "vvwind_zht_fcstfld"
)
```

`GetForecast(ctx, loc)`:
1. `row, col, err := i.rowCol(ctx, loc)` — GET latlon2rowcol, parse first point
2. `runDate, err := i.latestRunDate(ctx, row, col)` — GET date list, RLE-decode, pick latest
3. `uTimes, uData, err := i.fetchComponent(ctx, row, col, icmFieldU, runDate)` — POST forecast
4. `vTimes, vData, err := i.fetchComponent(ctx, row, col, icmFieldV, runDate)` — POST forecast
5. zip by index (times arrays expected identical — error if lengths differ), compute speed/angle per pair, build `[]WindDataPoint` keyed `"hourly"`, set `reading.Location = loc`

Helper signatures (unexported):
- `func (i *IcmMeteo) get(ctx context.Context, path string, out any) error` — shared GET+auth-header+JSON-decode, used by rowCol and latestRunDate
- `func (i *IcmMeteo) fetchComponent(ctx context.Context, row, col int, field, date string) ([]time.Time, []float64, error)` — POST + parse `{times, data}`, time layout `time.RFC3339` (docs say ISO 8601 UTC — confirm exact layout against a live response/fixture during implementation, adjust like `openMeteoTimeLayout`)
- RLE decode helper for the `dates` listing → latest run timestamp string in `yyyy-mm-ddTHH` form

### 2. Wiring — `main.go`

- Add env var `ICM_METEO_TOKEN` (mirror the `OPENWEATHER_TOKEN` check at `main.go:79-82`); add `ICM_METEO_TOKEN=<your icm.meteo.pl token>` to `.env.template`
- Construct once: `icmMeteo, err := internal.NewIcmMeteo("https://api.meteo.pl", icmToken)`
- Append to the providers slice at `main.go:105-109`: `{Name: "icm-meteo", Forecaster: icmMeteo}`

### 3. Tests — `internal/icmmeteo_test.go`

Mirror `internal/openmeteo_test.go` (httptest.Server, `reflect.DeepEqual` on `[]WindDataPoint`, fixtures in `testdata/`):
- Single mock server with a handler that switches on `r.URL.Path` / `r.Method` to serve 4 fixtures: rowcol response, date-list response, U-forecast response, V-forecast response
- Assert `Authorization: Token <token>` header present on every call
- Assert computed `WindSpeed`/`WindAngle` match hand-computed expected values from known U/V pairs (pick simple numbers, e.g. u=0,v=-5 → speed 5, angle 0)
- New fixtures: `testdata/icmmeteo_rowcol.json`, `testdata/icmmeteo_dates.json`, `testdata/icmmeteo_u.json`, `testdata/icmmeteo_v.json`
- Add a small table-driven unit test for the U/V → speed/angle conversion covering the four cardinal directions

### 4. Cost note to flag for the user (not a code change)

Each `GetForecast` call now issues **2 paid POST requests** (U + V) plus 2 free GETs (rowcol, date-list), per location, per Lambda invocation. With N stored locations this is `2N` paid calls per run — worth the user double-checking ICM's pricing/budget before deploying to the schedule that triggers this Lambda regularly.

## Verification

- `go test -v -run TestIcmMeteo ./internal/` — new provider unit tests pass
- `make test` — full suite incl. existing providers/aggregator/rule-engine still green
- `make lint`
- Manual: with a real `ICM_METEO_TOKEN` set in `.env`, `make run-docker` + `make run-test-request` (or `LOCAL_MODE=true go run .`) and confirm the rendered mail / stdout shows an `icm-meteo` entry in `MatchedBy`/provider counts without errors in the logs
