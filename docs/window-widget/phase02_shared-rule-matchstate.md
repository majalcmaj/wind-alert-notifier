<!-- plan-status: pending -->
# Phase 02 — shared-rule-matchstate

> **Status:** ⬜ PENDING

Read `docs/window-widget/prompt.md` first.

## Goal
One shared 3-state rule evaluation — `Match` / `NearMiss` / `NoMatch` — in a new
`shared/rules` package, used by the alert engine and (later) the widget payload. The
widget's amber "speed fits, direction doesn't" hatching and its match lanes are both
derived from this single function; nothing downstream re-implements rule logic.

## Red
Create `shared/rules/rules_test.go` with a table test (fails to compile — package
doesn't exist):

```go
func TestEvaluate(t *testing.T) {
	rule := model.Rule{
		SpeedRange: model.Range{From: 12, To: 25},
		AngleRange: model.Range{From: 270, To: 360},
		HourRange:  model.Range{From: 6, To: 20},
	}
	cases := []struct {
		name string
		dp   forecast.WindDataPoint // helper to build: speed, angle, hour
		want rules.MatchState
	}{
		{"all constraints met", dp(15, 300, 12), rules.Match},
		{"direction wrong, rest fits", dp(15, 120, 12), rules.NearMiss},
		{"speed too low", dp(8, 300, 12), rules.NoMatch},
		{"speed ok but outside hours", dp(15, 120, 3), rules.NoMatch}, // NOT a near-miss
		{"cyclic angle range wraps north", dpRule(320, model.Range{From: 315, To: 45}), rules.Match},
		{"cyclic hour range wraps midnight", ...},
	}
}
```

The fourth case is the semantic decision worth spelling out: **near-miss means every
constraint passes except direction.** A 3 a.m. datapoint with the right speed is not a
near-miss — otherwise night hours flood the widget's lanes with meaningless amber.

## Green
1. New package `shared/rules`:

```go
type MatchState int

const (
	NoMatch MatchState = iota
	NearMiss // direction is the only failing constraint
	Match
)

// Evaluate classifies one datapoint against one rule. Hour-of-day is taken
// from dp.Time in tz (time.UTC preserves today's behavior; phase 03 threads
// the location's zone through).
func Evaluate(rule model.Rule, dp forecast.WindDataPoint, tz *time.Location) MatchState
```

2. Move the range logic from `alert-job/internal/rule_engine.go`'s `matches()` into
   `Evaluate` (the cyclic checks already live on `model.Range` — reuse, don't copy).
3. Rewrite alert-job's `matches(r, dp)` as
   `rules.Evaluate(r, dp, time.UTC) == rules.Match`. `EvaluateWithConfidence` and
   `EvaluateForecast` stay in alert-job (they are mail-flow orchestration), now built
   on the shared primitive.

`make test` green, including the new table.

## Refactor
Remove the `Matched bool` field from `forecast.WindDataPoint` — a presentation flag
has no business in a shared domain type (it made the mail template's job easy at the
cost of muddying the model). Replace with an alert-job-local view type where the mail
model is built:

```go
type renderedDataPoint struct {
	forecast.WindDataPoint
	Matched bool
}
```

`EvaluateForecast` returns the set of matched datapoint indices (or the view slice)
instead of mutating shared state; `mail_renderer.go` consumes the view type. Template
unchanged. This is the phase's main clean-code payoff: shared types describe weather,
alert-job types describe the mail.

## Verify
- `make ci` green.
- Rule-engine behavior identical: existing `EvaluateWithConfidence` /
  `EvaluateForecast` tests pass unmodified (except mechanical type renames in test
  setup).
- `grep -rn "Matched" shared/` returns nothing.

## Commit
`feat(shared): add rules.Evaluate with three-state matching`
