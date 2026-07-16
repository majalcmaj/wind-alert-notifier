<!-- plan-status: pending -->
# Phase 07 — widget-particles

> **Status:** ⬜ PENDING

Read `docs/window-widget/prompt.md` first.

## Goal
The animated particle map sits above the timeline, driven by the same clock: scrubbing
changes the wind the particles ride; the readout chip shows time / knots / gusts /
direction / focused-rule verdict; ▶ plays through the forecast. Reduced-motion users
get a static field and no autoplay. This completes Concept D v2.

## Red
Vitest tests for the engine's pure parts (fail — module doesn't exist):

```ts
// particles.test.ts
expect(speedColor(5)).toBe(THEME.speedScale[0]);   // <8 kn
expect(speedColor(30)).toBe(THEME.speedScale[4]);  // ≥25 kn

// meteorological sign convention: wind FROM 0° (north) moves particles +y (south)
const p = stepParticle({x: 0, y: 0, dir: 0, spd: 10, ...}, 1);
expect(p.y).toBeGreaterThan(0);
expect(p.x).toBeCloseTo(0);

// respawned particles sample around the CURRENT cursor wind
const field = createField({width: 100, height: 100});
field.setWind({speedKn: 20, dirDeg: 90});
expect(spawnFor(field).dir).toBeGreaterThan(60); // within jitter of 90 ± 10
```

## Green
1. **Vendor the engine**: port `../wind-range-widget/src/wind-particles.ts` (MIT,
   same author) into `web/frontend/src/particles.ts` — do not rewrite from scratch;
   its resize handling (ResizeObserver) and lifecycle (`destroy()`) are already
   tested upstream. Apply exactly these deltas:
   - drive model: `setWind({speedKn, dirDeg})` replaces the 7 s dir-range sweep cycle;
   - velocity scales with wind: `pxPerSec = 18 + kn * 5` (original: constant 120);
   - streak length scales: `len = 8 + kn * 1.1`; color from the 5-step scale
     (original: constant gray);
   - keep the movement math verbatim — `x -= sin(rad)·v·dt; y += cos(rad)·v·dt` is
     the FROM-convention encoding; "fixing" the signs is the classic bug here.
   - **new**: honor `devicePixelRatio` (the mockups skipped it; production canvases
     look blurry on retina without it):

```ts
canvas.width = w * devicePixelRatio;
canvas.height = h * devicePixelRatio;
ctx.scale(devicePixelRatio, devicePixelRatio);
```

2. **Transition feel** — this is behavior, not decoration, so state it: particles
   read the wind only at spawn; with 1.4–2.6 s lifetimes the field drifts to a new
   scrub position over ~1.5 s instead of snapping. Do not add per-frame re-aiming.

3. **Map panel** (`map.ts`): water-gradient background + pulsing location marker.
   Coastline silhouette is explicitly out of scope; leave the seam
   `renderBackground(el: HTMLElement, loc: LocationView): void` with the gradient
   implementation, so a per-location SVG asset can slot in later without touching the
   engine.

4. **Readout chip** (`readout.ts`): clock-subscribed; time (in `location.timezone`),
   rounded knots + rotated arrow, gusts, compass + degrees, focused-rule verdict.
   Verdict text comes from `format.describeRuleState` — the same function the
   status line uses (phase 06); one formatter, two consumers.

5. **Play** (`playback.ts`): rAF loop advancing the clock at 4 forecast-hours per
   real second, looping at the end; any scrub stops playback. Button toggles ▶/⏸ with
   `aria-pressed`.

6. **Reduced motion**: `matchMedia("(prefers-reduced-motion: reduce)")` → no rAF
   loop, no autoplay; on each clock change reseed and draw one static frame. All
   information the animation carries is already in the readout as text.

7. Assemble in `main.ts`: map panel + controls row (play, hint, speed-scale legend) +
   timeline. The page shell template gets the final layout/styles (dark panel is
   self-contained CSS scoped under the widget root — the rest of the admin stays
   Pico-themed).

## Refactor
- `main.ts` should read as a table of contents: create clock → fetch payload → mount
  map, controls, timeline, statusline — under ~40 lines. Push anything else down.
- Kill duplicated constants: speed thresholds, colors, and the knots-per-px factors
  live only in `theme.ts` / `particles.ts` respectively.
- Run a bundle-size sanity check (`ls -la dist`): target well under 50 kB unminified;
  investigate anything bigger (accidental dependency).

## Verify
- `make ci` green.
- `make -C web frontend && make up`, open `/locations/sopot/forecast`:
  - scrub Sunday afternoon in the seeded story → particles turn SE and red-streaked,
    readout flips to the near-miss verdict;
  - ▶ plays and loops; scrub stops it;
  - devtools → emulate `prefers-reduced-motion: reduce` → static field, ▶ inert;
  - retina/devtools zoom: streaks crisp (DPR handling);
  - lane click re-aims the readout verdict without touching playback.
- Final side-by-side with `docs/concept-d-v2.html`; screenshot for the PR.

## Commit
`feat(web): particle map + shared clock playback — Concept D v2 complete`
