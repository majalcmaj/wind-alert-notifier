# Wind Alert — Improvement Ideas

> Related: [`forecast-page-concepts.html`](forecast-page-concepts.html) — three interactive
> UX mockups for a future forecast + history page (open directly in a browser).

Prioritized backlog of UX and capability improvements for wind-/kite-surfers.
Each item says **what** to build, **why** it matters to a surfer, **how** to implement it
in this codebase (concrete files, types, examples), and **acceptance criteria** so an
implementer can verify they are done.

Codebase orientation (see root `CLAUDE.md`):

- `shared/model` — `Location`, `Rule`, `Range` structs shared by both lambdas.
- `shared/dynamo` — DynamoDB `Store` (locations + rules tables).
- `alert-job/` — cron Lambda: fetches forecasts from 4 providers (OpenWeather, yr.no,
  Open-Meteo, ICM), evaluates rules with cross-provider confidence
  (`internal/rule_engine.go`), renders MJML email, sends via SES.
- `web/` — htmx admin UI Lambda: CRUD for locations and rules.

Priority tiers:

- **P0 — correctness**: current behavior silently wrong.
- **P1 — alert quality**: fewer false alerts, alerts a surfer can act on.
- **P2 — safety & conditions data**: information that decides go / no-go.
- **P3 — admin UX**: make the web UI pleasant and mistake-proof.
- **P4 — nice-to-have**: valuable but not urgent.

---

## P0 — Correctness

### 1. Hour rules match UTC hours, not local time

**Problem.** `alert-job/internal/rule_engine.go:17` matches a rule's `HourRange` against
`dp.Time.Hour()`. Providers return Unix timestamps, which Go parses into UTC. A rule
"wind between 06:00 and 20:00" for Sopot (CET/CEST) actually matches **07:00–21:00 or
08:00–22:00 local time** depending on daylight saving. The user thinks they set local
hours; they silently get UTC hours.

**Fix.**

1. Add `Timezone string` to `model.Location` (IANA name, e.g. `"Europe/Warsaw"`):

   ```go
   type Location struct {
       ID       string  `json:"id"       dynamodbav:"id"`
       Name     string  `json:"name"     dynamodbav:"name"`
       Lat      float64 `json:"lat"      dynamodbav:"lat"`
       Lon      float64 `json:"lon"      dynamodbav:"lon"`
       Timezone string  `json:"timezone,omitempty" dynamodbav:"timezone,omitempty"`
   }
   ```

2. In the rule engine, convert before extracting the hour. The engine currently has no
   access to the location, so thread the `*time.Location` through (e.g. resolve it once
   per location in `main.go` and pass it to `EvaluateWithConfidence` / `matches`):

   ```go
   local := dp.Time.In(tz) // tz := time.LoadLocation(loc.Timezone), fallback time.UTC
   hour := float64(local.Hour()) + float64(local.Minute())/60.0
   ```

3. Empty/invalid timezone → fall back to UTC and keep old behavior (backward compatible
   with existing DynamoDB items that lack the attribute).
4. Web UI: add a timezone field to the location form (`web/internal/web/templates/location_form.html`),
   validate with `time.LoadLocation` in `web/internal/validate/validate.go`.
5. Email: render datapoint times in the location's timezone too — same bug exists in
   the displayed table otherwise.

**Acceptance.** Unit test: rule `hour 6–20`, datapoint at `05:30 UTC`, location
`Europe/Warsaw` (UTC+2 in summer) → matches, because it is 07:30 local. Same datapoint
with empty timezone → does not match.

---

## P1 — Alert quality

### 2. Require N consecutive matching hours ("session window")

**Problem.** One matching datapoint triggers an alert. A single hour of good wind is not
a session — nobody rigs up for 45 minutes. Spiky forecasts cause false alerts.

**Fix.**

1. Add to `model.Rule`:

   ```go
   MinConsecutiveHours int `json:"min_consecutive_hours,omitempty" dynamodbav:"min_consecutive_hours,omitempty"`
   ```

   `0` (absent) = current behavior (1 datapoint is enough).
2. In `EvaluateForecast` (`rule_engine.go`), evaluate the **hourly** series in time
   order. Instead of "any datapoint matches", find the longest run of consecutive
   matching hourly datapoints; trigger only if run length ≥ `MinConsecutiveHours`.
   Example: rule needs 3 consecutive hours; hourly matches at 10:00, 11:00, 14:00 →
   longest run is 2 (10–11) → no alert. Matches at 10:00, 11:00, 12:00 → run of 3 → alert.
   Note the hourly slice key in `WeatherReading.Readings` (currently `"hourly"` /
   `"daily"` style keys from providers) — apply the run logic per contiguous series,
   and treat gaps in timestamps (> 1h apart) as breaking the run.
3. Only datapoints inside the winning run get `Matched = true` (so the email bolds the
   session, not stray single hours).
4. Web UI: add "Min consecutive hours" number input to
   `web/internal/web/templates/rule_form.html`, parse in `web/internal/server/rules.go`,
   validate `>= 0` in `validate.go`.
5. Include it in `Rule.Describe()` (`shared/model/model.go`), e.g.
   `"…, at least 3h in a row"`.

**Acceptance.** Unit tests for: run at start of series, run at end, gap breaks run,
`MinConsecutiveHours: 0` keeps old single-point behavior, daily datapoints unaffected.

### 3. Alert deduplication / cooldown

**Problem.** The job runs on a schedule (cron). If Saturday looks windy, every run from
Wednesday to Saturday re-sends essentially the same alert. Users learn to ignore the mail.

**Fix.**

1. New DynamoDB table `wind-alert-state` (Terraform, `terraform/dynamodb.tf`), PK
   `location_id`, SK `rule_name`, attributes `fingerprint` (string), `sent_at` (RFC3339),
   plus a TTL attribute so stale items expire (e.g. 7 days).
2. Fingerprint = hash of the matched **forecast window**, not the send time. Example:
   `sha256(location_id + rule_name + firstMatchedHour.Truncate(time.Hour).UTC().String())`.
   If the same rule fires for the same forecast window again → skip.
3. In `alert-job/main.go`, after `EvaluateWithConfidence`: load state for triggered
   rules, drop rules whose fingerprint is unchanged, send mail only if any rule
   survives, then write new fingerprints. `alert-job` needs write access to **only this
   table** — keep the "web is the only writer" invariant for the config tables; state
   table is owned by alert-job (update IAM in Terraform accordingly, and document it in
   `web/ARCH.md`).
4. Escape hatch: if confidence increased materially since last send (e.g. from 0.5 to
   1.0), re-alert with subject prefix "Update: ".

**Acceptance.** Two consecutive local runs (`make run-local` + `make run-test-request`)
with the same seeded forecast produce one mail, not two. Changing the matched window
produces a new mail.

### 4. Show wind in knots (and m/s)

**Problem.** Kitesurfers and windsurfers think in knots (kite sizes, spot lore, every
forecast app). Emails and forms are m/s only — users mentally multiply by 2 all the time.

**Fix.**

1. Template function in `alert-job/internal/mail_renderer.go`:

   ```go
   func knots(ms float64) string { return fmt.Sprintf("%.0f kn", ms*1.9438445) }
   ```

   Register next to `windArrow` in the template `FuncMap`.
2. In `internal/mail_template.mjml`, render both: `8.2 m/s (16 kn)`. Regenerate HTML
   with `make generate` (never hand-edit `mail_template.html`).
3. `Rule.Describe()` in `shared/model/model.go`: append knots —
   `"wind 5.0–15.0 m/s (10–29 kn) from NE–SE …"`.
4. Web UI rule form: display a live hint under speed inputs ("= 16 kn") with a few lines
   of vanilla JS; store m/s unchanged (no schema change, no unit-conversion bugs).

**Acceptance.** Mail renderer test asserts `"(16 kn)"` appears for an 8.23 m/s datapoint.

### 5. Per-provider spread table in the email

**Problem.** Confidence is computed (`ConfidentRule.MatchedBy`, `TotalProviders`) and the
first successful provider's reading is displayed (`alert-job/main.go:110-121`), but the
reader can't see **disagreement**. "3/4 providers agree" hides that the fourth predicts
half the wind.

**Fix.**

1. `LocationResult` already has access to all `ProviderReading`s at build time in
   `main.go` — add `AllReadings []ProviderReading` (or a pre-digested view struct) to
   `internal.LocationResult`.
2. In the MJML template, for the matched hours only, add a compact table:

   | Hour  | openweather | yrno | openmeteo | icm-meteo |
   |-------|-------------|------|-----------|-----------|
   | 14:00 | 9.1 m/s NW  | 8.4 m/s NW | 9.8 m/s NNW | — |

   "—" for providers that errored (already collected as `ProviderIssues`).
3. Show `Confidence` and `MatchedBy` per triggered rule (already on `ConfidentRule`,
   currently unused in the template): `"Strong NW afternoon (confidence 75%: openweather, yrno, openmeteo)"`.

**Acceptance.** Mail renderer test with two providers disagreeing renders both values.

### 6. Notification channels beyond email + configurable recipients

**Problem.** Recipient and sender are hardcoded in `alert-job/main.go:37-38`. Email is
slow-twitch; "wind is on NOW" wants push. Also blocks having more than one user.

**Fix (staged).**

1. **Stage 1 — unhardcode:** `ALERT_FROM`, `ALERT_TO` (comma-separated) env vars, set via
   Terraform variables. Fail at startup if unset, like `OPENWEATHER_TOKEN`.
2. **Stage 2 — Telegram:** a `telegramSender` implementing the existing `mailSender`-style
   interface (rename interface to `alertSender` with `send(ctx, subject, body)`); Telegram
   Bot API is a single `POST https://api.telegram.org/bot<token>/sendMessage` with
   `chat_id` + `text` (use a plain-text summary, not the HTML mail). Env:
   `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`. Send to all configured channels.
3. **Stage 3 — per-rule channel/recipient:** add optional `Notify []string` to
   `model.Rule` (e.g. `["email:foo@bar.com", "telegram:12345"]`) once multi-user is real.
   Skip until needed.

**Acceptance.** Stage 1: no literals in `main.go`; local run with `LOCAL_MODE=true`
unaffected. Stage 2: with only Telegram configured, alert arrives as Telegram message.

---

## P2 — Safety & conditions data

### 7. Wind gusts

**Problem.** `WindDataPoint` (`alert-job/internal/openweather.go:23`) has only
`WindSpeed` + `WindAngle`. Gusts are the single most important missing datapoint:
mean 8 m/s gusting 16 is dangerous and unrideable on a kite; steady 8 is perfect.
All four providers expose gusts.

**Fix.**

1. Add `WindGust float64` to `WindDataPoint`.
2. Map it per provider:
   - OpenWeather One Call: `wind_gust` field on hourly/daily.
   - yr.no (met.no locationforecast): `wind_speed_of_gust` in `instant.details`.
   - Open-Meteo: request `wind_gusts_10m` in `hourly=` params.
   - ICM: check response schema (`docs/meteo/api-doc.html`); if absent, leave 0 and
     exclude from gust-based matching for that provider.
3. Add optional `GustRange Range` to `model.Rule` (`json:"gust,omitempty"`). Semantics:
   zero-value range = ignore gusts (backward compatible). Typical use: `gust.to` as a
   cap — "alert only if gusts stay below 14 m/s". In `matches()`:

   ```go
   gustOK := r.GustRange == (model.Range{}) || r.GustRange.WithinRange(dp.WindGust)
   ```

4. Show gusts in the mail table as `8.2 (12.4) m/s` and in the provider-spread table.
5. Web UI: two optional inputs "Gust from / Gust to" in `rule_form.html`; empty = ignore.
6. A useful derived display: **gust factor** = gust/mean. Above ~1.6 = gusty/unpleasant.
   Show it, don't rule on it (keep rules simple).

**Acceptance.** Provider tests updated with gust fixtures; rule test: mean in range but
gust above cap → no match; rule without gust range → old behavior.

### 8. Onshore / offshore / cross-shore classification

**Problem.** Wind direction in degrees doesn't tell a kiter the one thing that decides
safety: **offshore wind can blow you out to sea**. Direction relative to the beach does.

**Fix.**

1. Add `ShoreBearing float64` to `model.Location` (with a `has value` convention, e.g.
   pointer or `-1` default): compass bearing **from land to water**, i.e. the direction
   you face standing on the beach looking at the water. Example: beach faces north
   (water to the north) → `ShoreBearing: 0`.
2. Classification helper in `shared/model`:

   ```go
   // WindShoreRelation classifies meteorological wind direction (degrees the wind
   // comes FROM) relative to the shore bearing (land→water direction).
   func WindShoreRelation(windFromDeg, shoreBearing float64) string {
       diff := math.Abs(math.Mod(windFromDeg-shoreBearing+540, 360) - 180)
       // diff is the angle between wind vector and the land→water vector.
       switch {
       case diff < 45:  return "offshore"      // wind blows land→water: DANGER
       case diff < 67.5: return "cross-off"
       case diff < 112.5: return "cross-shore"
       case diff < 135: return "cross-on"
       default:          return "onshore"
       }
   }
   ```

   (Verify the geometry with table-driven tests before trusting it: wind FROM 180°
   at a north-facing beach — water at 0° — blows from land to water = offshore; diff
   = 0 → offshore. Wind FROM 0° there = onshore; diff = 180 → onshore. Correct.)
3. Mail: badge per matched hour — `⚠ offshore` in red, `cross-on` etc. Plain text is
   fine; MJML supports inline styles.
4. Web UI: optional "Shore bearing" field on location form with helper text:
   "Compass direction from beach towards the water. North-facing beach = 0."
5. Later (optional): rule filter `AllowedShore []string` ("alert only when cross or on").

**Acceptance.** Table-driven test for `WindShoreRelation` covering all 5 classes and
wrap-around (windFrom 350°, shore 10°). Mail shows badge when `ShoreBearing` set,
nothing when unset.

### 9. Air temperature (and rain)

**Problem.** 12 m/s in April at 6°C with rain vs 12 m/s in July at 24°C are different
decisions (5/3 hooded wetsuit vs boardshorts). The mail shows wind only.

**Fix.**

1. Add `TempC float64` and `PrecipMM float64` to `WindDataPoint`; map from providers
   (OpenWeather: `temp`, `rain.1h`; Open-Meteo: `temperature_2m`, `precipitation`;
   yr.no: `air_temperature`, `precipitation_amount` on next-1h block; ICM per its schema).
2. Show both as columns in the hourly table. Rain > 0 → droplet emoji 💧 + amount.
3. Do **not** add them to rules yet — display only. Rules stay about wind; a human can
   read temperature.

**Acceptance.** Provider fixture tests assert temp/precip mapping; mail renders columns.

### 10. Waves and water temperature (Open-Meteo Marine)

**Problem.** For coastal spots, wave height/period decides gear and skill floor; water
temperature decides wetsuit. No provider currently fetched covers these.

**Fix.**

1. New provider-like client `internal/openmeteo_marine.go` calling
   `https://marine-api.open-meteo.com/v1/marine?latitude=…&longitude=…&hourly=wave_height,wave_period,wave_direction,sea_surface_temperature`.
   It is **not** a wind `Forecaster` — model it as a separate optional fetch returning
   `MarineReading` (hourly wave/SST series), attached to `LocationResult`.
2. Only fetch when the location is coastal — simplest: `Marine bool` flag on
   `model.Location`, set in the web UI ("Coastal spot: fetch waves & water temp").
3. Mail: add wave height (m), period (s) and SST columns for matched hours. Interpretive
   hint is welcome: period < 4 s = wind chop, > 8 s = swell.
4. Failure of the marine fetch must not block the wind alert — treat like a
   `ProviderIssue`.

**Acceptance.** Fixture test for the marine client; mail renders marine columns only when
flag set; marine API error still produces wind-only mail.

### 11. Thunderstorm risk

**Problem.** Squalls and lightning are the most dangerous conditions for kiters. A windy
afternoon with an approaching CB front should scream a warning, not look like a great session.

**Fix (minimal).**

1. OpenWeather One Call already returns `weather[].id` per hour; ids `2xx` are
   thunderstorms. Open-Meteo offers `weathercode` (95–99 = thunderstorm) and `cape`.
2. Add `Thunder bool` to `WindDataPoint`, set from those fields.
3. If any matched hour (or any hour ±2h around the matched window) has `Thunder`,
   prepend a prominent warning block to the mail: "⚡ Thunderstorm risk during or near
   the matched window — check radar before going." Never suppress the alert; inform.

**Acceptance.** Renderer test: thunder flag on a matched hour renders warning block.

---

## P3 — Admin UI UX

### 12. Dry-run: "test this rule against the live forecast"

**Problem.** The #1 admin question is "why did/didn't I get an alert yesterday?" There is
no way to see rule evaluation without CloudWatch spelunking.

**Fix.**

1. Extract rule evaluation so it is callable from `web`: the evaluation code
   (`FetchAll`, `EvaluateWithConfidence`, `WindDataPoint`) lives in
   `alert-job/internal` — not importable across modules. Move the rule engine +
   forecast-fetch interfaces into `shared/` (e.g. `shared/forecast`), keeping provider
   implementations where they are or moving them too. This is the main refactor cost.
2. New route `POST /locations/{id}/rules/{name}/test` in `web/internal/server/rules.go`.
   Handler: fetch forecasts for the location (requires the provider tokens as env vars
   on the web lambda too), evaluate just this rule, render a fragment: table of the next
   48 hourly datapoints with ✓/✗ per hour and per provider, plus resulting confidence.
3. "Test" button on each rule row (`rule_row.html`) with
   `hx-post="…/test" hx-target="closest tr" hx-swap="afterend"` showing the fragment inline.
4. Cheaper interim alternative (if the refactor is too much): a "run now (dry)" that
   invokes the alert-job lambda with a `dry_run` flag and shows its JSON response —
   the handler already returns evaluated results as JSON.

**Acceptance.** Clicking Test on a rule shows per-hour, per-provider match table without
sending any mail.

### 13. Live rule sentence preview in the form

**Problem.** Six numeric range inputs with wrap semantics; easy to enter a rule that
means something else than intended.

**Fix.** `Rule.Describe()` already generates the human sentence server-side. Add a small
`GET /rules/preview?angle_from=…&…` endpoint returning the sentence as a text fragment,
and wire the form inputs in `rule_form.html` with
`hx-get="/rules/preview" hx-trigger="input changed delay:300ms from:closest form" hx-include="closest form" hx-target="#rule-preview"`.
Shows e.g. **"wind 5.0–15.0 m/s (10–29 kn) from NE–SE (45°–135°), 06:00–20:00"** live
while typing. Zero JS. Also render the wrap hint dynamically: when `angle_from > angle_to`,
the sentence naturally shows the wrap ("from NW–NE (315°–45°)") which confirms intent.

**Acceptance.** Typing in any range input updates the sentence within ~300 ms.

### 14. Compass widget for the angle range

**Problem.** "Angle from 292.5, angle to 67.5" — nobody thinks like that; wrap ranges
are especially error-prone.

**Fix.** Small vanilla-JS + inline-SVG widget in `rule_form.html` (no CDN — assets are
embedded, consistent with the no-external-deps rule):

- SVG circle with N/E/S/W labels; the selected range drawn as an arc (two draggable
  handles, or click-to-set from/to).
- Widget writes to the existing hidden/visible `angle_from`/`angle_to` inputs; the form
  submits exactly as today, so the server side does not change at all.
- Arc rendering must handle wrap: if `from > to`, draw the arc through 360/0.
- Keep the numeric inputs visible and two-way-bound (type a number → arc updates).
  ~100–150 lines of JS; ship in `web/internal/web/static/compass.js`.

**Acceptance.** Selecting an arc across north (e.g. NW→NE) fills `from=315, to=45`;
form submits and rule round-trips correctly.

### 15. Map link + coordinate paste for locations

**Problem.** Getting lat/lon into two number fields means alt-tabbing to Google Maps and
copying numbers one by one.

**Fix (no CDN, so no embedded map tiles — two cheap wins instead):**

1. **Paste parsing:** one JS listener on the location form — pasting `54.646034, 18.512407`
   (Google Maps copy format) into the lat field splits and fills both fields.
2. **Verify link:** next to the fields, render
   `<a href="https://www.openstreetmap.org/?mlat={lat}&mlon={lon}#map=14/{lat}/{lon}" target="_blank">check on map</a>`
   (plain link out, not an embed — CSP/no-CDN constraint only applies to loaded assets).
   Update the link live as the fields change.

**Acceptance.** Pasting a Google-Maps coordinate string fills both fields; link opens OSM
at that position.

### 16. Alert history page

**Problem.** No record of what was sent. "Did it fire on Saturday?" is unanswerable.

**Fix.**

1. Reuse (or extend) the `wind-alert-state` table from item 3, or a dedicated
   `wind-alert-history` table: PK `location_id`, SK `sent_at#rule_name`, attributes:
   rule snapshot (`Describe()` output), confidence, matched providers, matched window,
   TTL 90 days. alert-job writes one item per triggered rule per sent mail.
2. Web: `GET /history` page + nav link in `layout.html` — reverse-chronological table:
   when, location, rule sentence, confidence, window. Query per location, merge, sort.
3. Read-only; no forms.

**Acceptance.** After a triggered local run, `/history` lists the alert with correct data.

### 17. "Run now" button

**Problem.** After editing rules, the only way to see the effect is waiting for the next
cron tick.

**Fix.** Button in the web UI header ("Evaluate now") → new route `POST /run` → invokes
the alert-job lambda via `aws-sdk-go-v2/service/lambda` `Invoke` (function name from env
var `ALERT_JOB_FUNCTION`; add `lambda:InvokeFunction` for that ARN to the web lambda's
IAM policy in Terraform). Show the JSON body of the response in a `<pre>` fragment.
Locally, this can call the alert-job container's RIE endpoint instead (env-switched URL).

**Acceptance.** Click sends real evaluation, response summary displayed; permission
scoped to the single function ARN.

### 18. Webcam / spot-info links per location

**Problem.** Forecast says go; surfer still checks the spot webcam. Make the alert
one-click actionable.

**Fix.** Add `Links []Link` (`{Label, URL string}`) to `model.Location`, editable as a
textarea in the location form (one `label|url` per line, parsed server-side). Render as
a link row in the mail under each location header and on the location row in the UI.
Validate URLs are `http(s)://`.

**Acceptance.** Location with a webcam link → link appears in triggered mail.

---

## P4 — Nice-to-have / later

### 19. Live observations to sanity-check the forecast

Nearest live station reading (Holfuy `api.holfuy.com`, or IMGW public data for Poland)
rendered in the mail: "Station Sopot Pier now: 6.2 m/s NW". Needs per-location station id
(`StationID string` on `Location`). Great trust-builder; also the seed of item 20.
Failure handling like any `ProviderIssue`.

### 20. Provider accuracy tracking → weighted confidence

Store forecast (provider, hour, predicted speed) vs. observed (from item 19) and compute
per-provider error per location. Replace the flat `matchedProviders/totalProviders`
confidence in `EvaluateWithConfidence` with weights. Requires months of data; build
the recording first, the weighting later.

### 21. Lead-time trend alerts

Alert early at low confidence ("Saturday might work — 3 days out"), then send "Update:"
mails as confidence changes materially (ties into fingerprint/cooldown from item 3 —
the fingerprint gains a confidence bucket). Surfers plan weekends; a Friday-evening
alert for Saturday morning is often too late.

### 22. Kite/sail size hint

Given rider weight (env var or per-recipient setting) and mean wind, suggest kite size
in the mail ("~9 m² for 75 kg @ 16 kn"). Simple lookup table is enough; label it as a
rough hint. Pure renderer change once knots (item 4) exists.

### 23. Multi-user / per-rule recipients

Stage 3 of item 6. Only when someone other than the owner actually subscribes.

---

## Suggested implementation order

| Order | Item | Why first |
|-------|------|-----------|
| 1 | #1 timezone | Silent correctness bug; small, self-contained |
| 2 | #4 knots | Trivial, immediate daily value |
| 3 | #7 gusts | Biggest data gap; touches all providers, do early before more rule fields |
| 4 | #2 session window | Kills the worst false alerts |
| 5 | #3 dedup/cooldown | Kills the second-worst annoyance; needs new table |
| 6 | #13 rule preview | Cheapest UI win, reuses `Describe()` |
| 7 | #5 provider spread | Renderer-only, data already in memory |
| 8 | #8 onshore/offshore | High safety value, moderate effort |
| 9 | #9 temp/rain | Display-only, low risk |
| 10 | #16 history + #17 run-now | Admin observability pair |
| 11 | #12 dry-run | Highest admin value but needs the shared-module refactor |
| 12 | #10 marine, #11 thunder, #14 compass, #15 map, #18 links | Independent, any order |
| 13 | #6 channels, then P4 items | As appetite allows |

General implementation notes for all items:

- Cross-package struct literals must use keyed fields (`go vet` composites check).
- New `model.Rule`/`model.Location` fields must be optional (`omitempty`, zero-value =
  old behavior) — existing DynamoDB items lack them.
- `make ci` (vet + test) must pass; run `make lint` for alert-job and web.
- Mail template changes: edit `internal/mail_template.mjml`, regenerate with
  `make generate`, commit both files.
- Web templates are embedded via `//go:embed` — new static assets go in
  `web/internal/web/static/` and are served under `/static/`.
