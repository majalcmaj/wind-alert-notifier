<!-- plan-status: pending -->
# Phase 03 — location-timezone

> **Status:** ⬜ PENDING

Read `docs/window-widget/prompt.md` first.

## Goal
`model.Location` carries an optional IANA timezone; rule hours evaluate in the
location's local time (fixing the standing bug where "06–20" means UTC hours), and the
widget payload can label hours correctly. Zero-value locations keep UTC behavior.

## Red
Two failing tests:

1. `shared/model/model_test.go`:

```go
func TestLocationTZ(t *testing.T) {
	warsaw := model.Location{Timezone: "Europe/Warsaw"}
	if warsaw.TZ().String() != "Europe/Warsaw" { t.Fatal(...) }
	if (model.Location{}).TZ() != time.UTC { t.Fatal("empty must fall back to UTC") }
	if (model.Location{Timezone: "Mars/Olympus"}).TZ() != time.UTC { t.Fatal("invalid must fall back, not error") }
}
```

2. `shared/rules/rules_test.go`: rule `hour 6–20`, datapoint at `05:30 UTC`, zone
   `Europe/Warsaw` (UTC+2 in July) → `Match`, because it is 07:30 local. Same
   datapoint with `time.UTC` → `NoMatch`. (Signature exists since phase 02; this
   pins the semantics.)

Both fail: no `Timezone` field, no `TZ()` method.

## Green
1. `shared/model`:

```go
type Location struct {
	// ... existing fields, keyed literals everywhere ...
	Timezone string `json:"timezone,omitempty" dynamodbav:"timezone,omitempty"`
}

// TZ resolves the location's zone, falling back to UTC. Invalid values never
// error at read time — the write path validates, the read path degrades.
func (l Location) TZ() *time.Location
```

`omitempty` keeps existing DynamoDB items valid — absent attribute = UTC = today's
behavior.

2. `web/internal/validate`: `ValidateLocation` accepts empty, rejects strings that
   `time.LoadLocation` refuses. Error message names the expected format:
   `"timezone must be an IANA name like Europe/Warsaw"`.
3. `web` location form (`location_form.html`): one text input with a `<datalist>` of
   a few common zones (Europe/Warsaw, UTC, Europe/Berlin) — a full tz dropdown is
   waste. Parse in `locations.go` handler alongside lat/lon.
4. `alert-job/main.go`: pass `loc.TZ()` where the evaluation loop calls into the rule
   engine (thread through `EvaluateWithConfidence` → `rules.Evaluate`).

Lambda note for the executor: `time.LoadLocation` needs tzdata. Both lambdas run in
scratch-style Docker images — import `_ "time/tzdata"` in each `main.go` (embeds the
zone database in the binary, ~450 kB) rather than installing OS tzdata packages.

## Refactor
- The mail template shows datapoint times: render them via `dp.Time.In(loc.TZ())` in
  the mail model builder so the email and the future widget agree on wall-clock time.
  One conversion site (the view builder), not scattered `.In()` calls.
- Check `Describe()` output still reads correctly (it prints rule hours, which are
  zone-agnostic numbers — no change expected; confirm, don't assume).

## Verify
- `make ci` green.
- `make up && make seed`, edit seeded location, set `Europe/Warsaw`, confirm it
  round-trips through the form and DynamoDB (`aws dynamodb scan` against local or the
  edit form re-shows it).
- Existing locations without the attribute still load (seed data has none — the list
  page renders).

## Commit
`feat(shared,web): per-location IANA timezone; rule hours evaluate in local time`
