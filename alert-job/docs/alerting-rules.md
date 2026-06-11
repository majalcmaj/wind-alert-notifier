# Wind Alert — Conditional Alerting Rules

## Context

Today `main.go` **always** sends the forecast email on every invocation. A
`RunRuleEngine` exists in `internal/rule_engine.go` but is dead code — only
called from tests, never wired into the handler, and no rules are defined
anywhere.

Goal: send the mail **only** when the forecast actually meets wind criteria
(hours / strength / direction). When triggered, still show the full forecast,
but **bold the datapoints that fired** and list, in plain language, **which
rules triggered**.

Decisions (confirmed with user):
- Rules live in an **embedded JSON file** (`rules.json`, `//go:embed`).
- Evaluate **both** hourly and daily datapoints.
- Mail = full forecast, with matching rows bolded + a "triggered rules" summary.

## Changes

### 1. Rule type — JSON tags + description — `internal/rule_engine.go`
- Add JSON tags to `Range` (`from`/`to`) and `Rule` (`angle`/`speed`/`hour`).
- Add optional `Name string `json:"name,omitempty"`` to `Rule`.
- Add `func (r Rule) Describe() string` — returns `Name` if set, else a generated
  sentence from the ranges, e.g.
  `"wind 5.0–15.0 m/s from NE–SE (45°–135°), 06:00–20:00"`.
  Reuse compass-direction logic akin to `renderWindArrow`
  (`internal/wind_arrow_renderer.go`) for the direction words; format hours from
  the float HourRange.
- Refactor the match condition out of `RunRuleEngine` into
  `func (r Rule) matches(dp WindDataPoint) bool` (the existing
  AngleRange/SpeedRange/HourRange check). `RunRuleEngine` keeps working via
  `matches` — existing tests stay green.

### 2. Mark matches + collect triggered rules — `internal/rule_engine.go` + `internal/types.go`
- `internal/types.go`: add `Matched bool` to `WindDataPoint` (internal-only flag;
  not part of OpenWeather JSON, set during evaluation).
- New: `func EvaluateForecast(reading *WeatherReading, rules []Rule) []Rule`
  - Iterate every datapoint in `reading.Readings` (hourly + daily).
  - For each datapoint, for each rule: if `rule.matches(dp)`, set
    `dp.Matched = true` and record the rule as triggered.
  - Return the **deduplicated, ordered** slice of triggered rules (empty = no alert).

### 3. Rules data — `internal/rules.go` (new) + `internal/rules.json` (new)
- `rules.json`: a JSON array of rules with sane Gdańsk/Sopot seed values, each
  with a `name`. Example entry:
  ```json
  { "name": "Strong NW afternoon wind",
    "speed": { "from": 6, "to": 25 },
    "angle": { "from": 270, "to": 360 },
    "hour":  { "from": 12, "to": 20 } }
  ```
- `rules.go`:
  ```go
  //go:embed rules.json
  var rulesJSON []byte
  var AlertRules []Rule
  func init() { /* json.Unmarshal(rulesJSON, &AlertRules); panic on error */ }
  ```

### 4. Mail model + renderer — `internal/mail_renderer.go`
- New view struct:
  ```go
  type MailModel struct {
      Reading        *WeatherReading
      TriggeredRules []string
  }
  ```
- Change `RenderMail` to accept `MailModel` (was `*WeatherReading`).

### 5. Template — `internal/mail_template.mjml` (then regenerate `.html`)
- Field paths shift under `.Reading` (`.Reading.Lat`, `.Reading.Readings.daily`…).
- Add a "Triggered rules" block above the tables:
  `{{range .TriggeredRules}}<li>{{.}}</li>{{end}}`.
- Per row, bold when matched:
  `{{if .Matched}}<strong>…</strong>{{else}}…{{end}}` (both daily + hourly tables).
- Regenerate with `make generate` (npx mjml). **The committed `.html` is what's
  embedded — must regenerate, not hand-edit.**

### 6. Wire into handler — `main.go`
After `GetForecast`:
```go
triggered := internal.EvaluateForecast(forecast, internal.AlertRules)
if len(triggered) == 0 {
    return &events.APIGatewayProxyResponse{StatusCode: 200,
        Body: "No rule matched — no mail sent"}, nil   // skip SES entirely
}
descs := make([]string, len(triggered))
for i, r := range triggered { descs[i] = r.Describe() }
mail, err := internal.RenderMail(internal.MailModel{Reading: forecast, TriggeredRules: descs})
```
Send SES only on the trigger path.

## Tests
- `internal/rule_engine_test.go`: keep existing `RunRuleEngine` tests. Add
  `EvaluateForecast` tests: no match → empty + no `Matched` flags; one match →
  correct datapoint flagged + rule returned; dedupe when a rule matches multiple
  points; matches across both hourly and daily.
- Add `Rule.Describe()` test (named rule returns name; unnamed generates sentence).
- `internal/mail_renderer_test.go`: update to new `MailModel` signature; assert
  bold markup appears for matched rows and triggered-rule text renders.

## Verification
1. `make test` — all green.
2. `make generate` — `.html` regenerates without diff surprises.
3. `make lint`.
4. Local end-to-end: `make build-docker && make run-docker`, then
   `make run-test-request`. With seed `rules.json`, confirm response body says
   either mail-sent or "No rule matched" depending on live forecast.
