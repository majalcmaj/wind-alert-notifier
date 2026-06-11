# Plan: Multi-Provider Forecast with Confidence Scoring

## Context

Wind alerts currently rely solely on OpenWeather One Call 3.0. A single source gives no signal when the model is uncertain, and a missed alert if that model is wrong. Adding yr.no (MET Norway) and Open-Meteo (multi-model NWP aggregator) provides independent forecasts. Confidence scoring — how many providers agree a rule fires — surfaces forecast certainty in the mail and lets users set a minimum agreement threshold per rule.

Both new providers are free, require no API key, and cover Gdańsk well.

---

## Data Flow

```
main.go
  ├── FetchAll(ctx, loc, []NamedForecaster{openweather, yrno, openmeteo})
  │     ├── goroutine: openweather.GetForecast(ctx, loc)  ─┐
  │     ├── goroutine: yrno.GetForecast(ctx, loc)          ├── []ProviderReading
  │     └── goroutine: openmeteo.GetForecast(ctx, loc)    ─┘
  │
  ├── EvaluateWithConfidence([]ProviderReading, rules)
  │     └── for each rule: EvaluateForecast(pr.Reading, rules) per provider
  │         → Confidence = triggeredCount / successfulProviders
  │         → []ConfidentRule
  │
  └── LocationResult{
        Reading:        readings[0].Reading,  // OpenWeather (primary, has hourly+daily)
        TriggeredRules: []ConfidentRule,
      }
        → RenderMail → SES
```

---

## Implementation Phases

### Phase 1 — Add `ctx` to `GetForecast` + define `Forecaster` interface

**`internal/openweather.go`**
- Change `GetForecast(location Location)` → `GetForecast(ctx context.Context, loc Location)`
- Replace `http.NewRequest` with `http.NewRequestWithContext(ctx, ...)`

**`internal/aggregator.go`** (new file — interface lives where it's consumed)
```go
type Forecaster interface {
    GetForecast(ctx context.Context, loc Location) (*WeatherReading, error)
}

type NamedForecaster struct {
    Name      string
    Forecaster Forecaster
}

type ProviderReading struct {
    Name    string
    Reading *WeatherReading
    Err     error
}
```

**`internal/openweather_test.go`** — pass `context.Background()` to updated call.  
**`main.go`** — pass `ctx` to `openWeather.GetForecast(ctx, loc)`.

---

### Phase 2 — yr.no adapter

**`internal/yrno.go`** (new)

Endpoint: `GET https://api.met.no/weatherapi/locationforecast/2.0/compact?lat={lat}&lon={lon}`  
Required header: `User-Agent: wind-alert-go/1.0 github.com/majalcmaj/wind-alert-go`  
Response path: `properties.timeseries[].time` + `.data.instant.details.wind_speed` + `.wind_from_direction`  
Maps to `WeatherReading.Readings["hourly"]` (no daily; yr.no is hourly only).

```go
type YrNo struct{ baseURL string }
func NewYrNo() *YrNo
func (y *YrNo) GetForecast(ctx context.Context, loc Location) (*WeatherReading, error)
```

**`internal/yrno_test.go`** (new) — `httptest.NewServer` + fixture at `testdata/yrno.json`, same pattern as `openweather_test.go`. Tests URL path, User-Agent header, and parsed `WeatherReading` fields.

---

### Phase 3 — Open-Meteo adapter

**`internal/openmeteo.go`** (new)

Endpoint: `GET https://api.open-meteo.com/v1/forecast?latitude={lat}&longitude={lon}&hourly=wind_speed_10m,wind_direction_10m&wind_speed_unit=ms&forecast_days=7`  
No auth required.  
Response: `hourly.time[i]` (ISO8601), `hourly.wind_speed_10m[i]`, `hourly.wind_direction_10m[i]`  
Maps to `WeatherReading.Readings["hourly"]`.

```go
type OpenMeteo struct{ baseURL string }
func NewOpenMeteo() *OpenMeteo
func (o *OpenMeteo) GetForecast(ctx context.Context, loc Location) (*WeatherReading, error)
```

**`internal/openmeteo_test.go`** (new) — same httptest pattern, fixture at `testdata/openmeteo.json`.

---

### Phase 4 — `FetchAll` parallel aggregation

**`internal/aggregator.go`** (extend)

```go
func FetchAll(ctx context.Context, loc Location, providers []NamedForecaster) []ProviderReading
```

- Spawns one goroutine per provider using `sync.WaitGroup`
- Collects into `[]ProviderReading` (order matches input slice order)
- Provider errors are recorded in `ProviderReading.Err` — non-fatal; degrade confidence

Ordering matters: `main.go` passes OpenWeather first; `readings[0]` is always the primary display source.

---

### Phase 5 — `Rule.MinConfidence` + `EvaluateWithConfidence`

**`internal/rule_engine.go`** (extend; keep existing `EvaluateForecast` unchanged)

Add to `Rule`:
```go
MinConfidence float64 `json:"min_confidence,omitempty" dynamodbav:"min_confidence,omitempty"`
```
Zero value = trigger on any single provider. Existing DynamoDB items unchanged.

New types:
```go
type ConfidentRule struct {
    Rule
    Confidence float64  // triggeredProviders / successfulProviders
    MatchedBy  []string // provider names that triggered this rule
}
```

New function:
```go
func EvaluateWithConfidence(readings []ProviderReading, rules []Rule) []ConfidentRule
```
Logic:
1. Count `successfulProviders` = providers where `Err == nil`
2. For each rule, call `EvaluateForecast(pr.Reading, []Rule{rule})` per successful provider
3. Collect provider names that triggered → `MatchedBy`
4. `Confidence = len(MatchedBy) / successfulProviders`
5. Include rule if `Confidence >= rule.MinConfidence` (or `MinConfidence == 0`)

The `Matched` flags on each provider's `Reading` are set in-place by `EvaluateForecast` — only `readings[0].Reading` (OpenWeather) is used for display, so its flags reflect OpenWeather's own evaluation.

---

### Phase 6 — Wire `main.go`

Replace:
```go
forecast, err := openWeather.GetForecast(loc)
triggered := internal.EvaluateForecast(forecast, locRules)
descs := make([]string, len(triggered))
for i, r := range triggered {
    descs[i] = r.Describe()
}
results = append(results, internal.LocationResult{..., TriggeredRules: descs})
```

With:
```go
providers := []internal.NamedForecaster{
    {"openweather", openWeather},
    {"yrno", internal.NewYrNo()},
    {"openmeteo", internal.NewOpenMeteo()},
}
readings := internal.FetchAll(ctx, loc, providers)
triggered := internal.EvaluateWithConfidence(readings, locRules)
if len(triggered) == 0 {
    continue
}
var displayReading *internal.WeatherReading
for _, pr := range readings {
    if pr.Err == nil {
        displayReading = pr.Reading
        break
    }
}
results = append(results, internal.LocationResult{
    Location:       loc,
    Reading:        displayReading,
    TriggeredRules: triggered,
})
```

Update `LocationResult` in `internal/mail_renderer.go`:
```go
type LocationResult struct {
    Location       Location
    Reading        *WeatherReading
    TriggeredRules []ConfidentRule  // was []string
}
```

---

### Phase 7 — Mail template

**`internal/mail_template.mjml`** — update triggered rules list:
```
{{range .TriggeredRules}}
<li>{{.Describe}} <span>({{.MatchedBy | join ", "}} — {{printf "%.0f" (mul .Confidence 100)}}%)</span></li>
{{end}}
```

Since Go templates don't have `join` built-in, add a `join` template func in `mail_renderer.go` alongside the existing `windArrow` func, or use a method on `ConfidentRule`:

```go
func (cr ConfidentRule) ConfidenceLabel() string {
    return fmt.Sprintf("%d/%d providers", len(cr.MatchedBy), /* total computed elsewhere */)
}
```

Simpler: add `ProvidersLabel() string` to `ConfidentRule` that returns e.g. `"2/3 providers"`. Store total provider count in `ConfidentRule` or compute from `Confidence` × total. Easiest: store `TotalProviders int` in `ConfidentRule`.

Template becomes:
```
{{range .TriggeredRules}}<li>{{.Describe}} ({{len .MatchedBy}}/{{.TotalProviders}} providers)</li>{{end}}
```

Regenerate HTML: `make generate` (requires `npx`).

**`internal/mail_renderer_test.go`** — update `TestRenderingTriggeredRules` to pass `[]ConfidentRule` instead of `[]string`.

---

## Files Modified

| File | Change |
|------|--------|
| `internal/openweather.go` | Add `ctx` parameter; use `NewRequestWithContext` |
| `internal/openweather_test.go` | Pass `context.Background()` to updated signature |
| `internal/rule_engine.go` | Add `MinConfidence` to `Rule`; add `ConfidentRule`, `EvaluateWithConfidence` |
| `internal/rule_engine_test.go` | Tests for `EvaluateWithConfidence` (2-of-3 scenarios) |
| `internal/aggregator.go` | **New** — `Forecaster` interface, `NamedForecaster`, `ProviderReading`, `FetchAll` |
| `internal/yrno.go` | **New** — `YrNo` implementing `Forecaster` |
| `internal/yrno_test.go` | **New** — httptest mock + `testdata/yrno.json` fixture |
| `internal/openmeteo.go` | **New** — `OpenMeteo` implementing `Forecaster` |
| `internal/openmeteo_test.go` | **New** — httptest mock + `testdata/openmeteo.json` fixture |
| `internal/mail_renderer.go` | Update `LocationResult.TriggeredRules` type to `[]ConfidentRule` |
| `internal/mail_renderer_test.go` | Update `TestRenderingTriggeredRules` to use `[]ConfidentRule` |
| `internal/mail_template.mjml` | Show rule name + provider count per triggered rule |
| `internal/mail_template.html` | Regenerated from mjml via `make generate` |
| `main.go` | Wire all three providers; use `FetchAll` + `EvaluateWithConfidence` |

---

## TDD Order

1. `openweather_test.go` — update to pass `ctx` → fails → add ctx to `GetForecast` → green
2. `yrno_test.go` → fails → implement `yrno.go` → green
3. `openmeteo_test.go` → fails → implement `openmeteo.go` → green
4. Test `FetchAll` with mock `Forecaster` implementations → implement → green
5. Test `EvaluateWithConfidence` with 2-of-3 scenario, 0-of-3, and `MinConfidence` filtering → implement → green
6. `make test && make build` before each commit

---

## Verification

1. `make test` — all tests pass
2. `make build` — compiles with `-tags lambda.norpc`
3. `make run-docker && make run-test-request` — Lambda logs which providers responded; response JSON includes `MatchedBy` and `Confidence` per triggered rule
4. Mail output (captured via `LOCAL_MODE=true`) shows provider counts next to each triggered rule
