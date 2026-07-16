<!-- plan-status: pending -->
# Phase 06 — widget-timeline

> **Status:** ⬜ PENDING

Read `docs/window-widget/prompt.md` first.

## Goal
The production timeline renders from the payload: speed plot with provider envelope,
direction-arrow strip, amber near-miss washes, per-rule match lanes with tap-to-focus,
draggable cursor, legend, and per-rule status line. All pure logic lives in small
named modules with vitest coverage; DOM code is thin orchestration.

Reference for every visual decision: `docs/concept-d-v2.html` (open it in a browser
next to the code — it is the spec).

## Red
Vitest tests for the pure cores, written first (all fail — modules don't exist):

```ts
// segments.test.ts — the engine behind lanes AND washes
expect(consecutiveRanges([0, 2, 2, 1, 2], s => s === 2)).toEqual([[1, 3], [4, 5]]);
expect(consecutiveRanges([], () => true)).toEqual([]);

// scales.test.ts
const sc = createScales({hours: 48, width: 920, maxKn: 35, margins});
expect(sc.x(0)).toBe(margins.left);
expect(sc.y(35)).toBe(margins.top);

// format.test.ts — timezone is the trap: NEVER Date.getHours()
expect(formatHour("2026-07-12T05:30:00Z", "Europe/Warsaw")).toBe("07:30");
expect(compassPoint(313)).toBe("NW");
expect(arrowRotation(0)).toBe(180); // wind FROM north blows south → glyph points down

// clock.test.ts
const clock = createClock(48);
const seen: number[] = [];
clock.subscribe(t => seen.push(t));
clock.set(99); expect(clock.get()).toBe(47); // clamped
```

`arrowRotation` deserves its one-line doc: meteorological direction says where wind
comes FROM; the glyph points where it blows TO, hence `(dirFrom + 180) % 360`.

## Green
Module layout — the clean-code centerpiece. Every layer is a function of
`(parent, view, scales)`, no globals, no classes where a closure does:

```
web/frontend/src/
  main.ts            mount(): fetch payload → assemble widget
  payload.ts         ForecastView types (phase 05)
  clock.ts           createClock(maxHours): { get, set, subscribe } — the ONLY mutable state
  scales.ts          createScales(): { x(t), y(kn), hourWidth }
  segments.ts        consecutiveRanges(states, predicate): [start, end)[]
  format.ts          formatHour, compassPoint, arrowRotation, describeRuleState
  svg.ts             svgEl(tag, attrs, children?) — the 15-line typed helper
  timeline/
    index.ts         createTimeline(root, view): { onFocusChange } — orchestrates layers,
                     owns focus state, subscribes to the clock for the cursor
    plot.ts          grid, envelope polygon, median line, rule band (patterned)
    washes.ts        near-miss columns for the focused rule (uses segments.ts)
    arrows.ts        direction strip; amber when focused rule state === NearMiss
    lanes.ts         one lane per rule from hourlyState (uses segments.ts); emits clicks
    cursor.ts        drag/pointer handling → clock.set; renders the cursor line
    patterns.ts      <defs>: dot/45°/135° rule patterns + amber near-miss hatch
  statusline.ts      per-rule verdict text under the timeline, clock-subscribed
```

Principles the executor must follow (they are the point of this plan):

- **One direction of data flow**: payload → view structs → layers. Interaction goes
  the other way only through two channels: `clock.set(t)` and `onFocusChange(ruleIdx)`.
  A focus change re-renders `plot`/`washes`/`arrows` layer groups; a clock tick moves
  only the cursor and text — never re-renders layers.
- **`hourlyState` comes from the payload.** The TS side never re-evaluates rules; if a
  lane looks wrong, the bug is in Go, findable by one language's tests.
- **No innerHTML string assembly** (the mockups did it; production doesn't). `svgEl`
  everywhere; layer functions stay under ~60 lines or split by name.
- Pointer handling essentials: `setPointerCapture` on pointerdown, `touch-action: none`
  on the SVG (without it, mobile browsers steal the drag for scrolling), lane hit
  rects swallow clicks before the drag handler
  (`if (target.closest("[data-lane]")) return`).
- Coordinates: convert client px → viewBox units via
  `(clientX - rect.left) / rect.width * VIEWBOX_WIDTH` — the SVG is responsive, the
  math is not optional.
- Legend is data-driven from the same constants the layers use (pattern ids, colors) —
  a legend that can drift from the chart is two sources of truth.
- Colors/patterns as exported constants in one `theme.ts` (amber `#fab219` near-miss,
  cyan accent, rule palette) — matching the reference design's values.

## Refactor
- Delete the phase-05 placeholder rendering from `main.ts`.
- Hunt duplication between `washes.ts` and `lanes.ts` — both are
  "ranges of a predicate over hourlyState drawn as rects"; if they converge, extract
  `renderStateRects(parent, ranges, style)` and keep the two callers one line each.
- Every exported function gets a one-sentence doc comment; nothing else needs comments
  if names are right — rename until that is true.

## Verify
- `make ci` green (vitest suite now ~15+ tests).
- `make -C web frontend && make up`: on `/locations/sopot/forecast` — drag scrubs,
  lane click refocuses (band + washes + arrows follow), legend matches visuals,
  status line updates, no console errors. Compare side-by-side against
  `docs/concept-d-v2.html`.
- Touch check in devtools device mode: drag works, page doesn't scroll while scrubbing.

## Commit
`feat(web): timeline widget — plot, arrows, near-miss washes, match lanes`
