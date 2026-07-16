<!-- plan-status: pending -->
# Phase 05 — frontend-scaffold

> **Status:** ⬜ PENDING

Read `docs/window-widget/prompt.md` first.

## Goal
A TypeScript workspace at `web/frontend/` (vite build, vitest tests, strict tsconfig)
producing one committed ESM bundle `web/internal/web/static/dist/window-widget.js`
that the phase-04 page loads; `make ci` runs the TS tests; the TS payload types are
contract-tested against the Go golden fixture.

## Red
Two failing checks:

1. `web/internal/web/web_test.go`: assert the embedded FS contains
   `static/dist/window-widget.js` — fails, file doesn't exist.
2. `cd web/frontend && pnpm test` — fails, workspace doesn't exist.

## Green
1. Scaffold `web/frontend/` (copy conventions from `../wind-range-widget`:
   `tsconfig.json` strict, vitest, pnpm). Keep `package.json` minimal — vite, vitest,
   typescript, nothing else. No UI framework.
2. `vite.config.ts` — lib mode, single deterministic artifact (committed files must
   not churn):

```ts
build: {
  lib: { entry: "src/main.ts", formats: ["es"], fileName: () => "window-widget.js" },
  outDir: "../internal/web/static/dist",
  emptyOutDir: true,
  minify: false,        // committed artifact stays reviewable in diffs
  sourcemap: false,
}
```

`minify: false` is a deliberate call: the bundle is code under review like any other
committed file, and it is served to one admin, not the public internet.

3. `src/payload.ts` — TS mirror of Go's `ForecastView` (exact field names). Contract
   test parses the shared golden fixture:

```ts
import fixture from "../fixtures/forecast-payload.json";
test("payload fixture satisfies ForecastView", () => {
  const view: ForecastView = fixture; // compile-time check is the assertion
  expect(view.aggregate.medianKn.length).toBe(view.hours.length);
});
```

Go writes this fixture (phase 04 golden test), TS reads it — drift in either language
breaks a test.

4. `src/main.ts` — minimal mount proving the pipeline:

```ts
export function mount(root: HTMLElement): void
// reads data-payload-url, fetches, renders "<n> h of forecast, <m> providers" text
```

Auto-mount on `#window-widget` at module load (the page shell from phase 04 already
has the element + script tag; adjust the template if the name differs).

5. **Build wiring** — repo convention is committed generated artifacts regenerated
   via make (see `alert-job`'s `make generate` for the MJML template):
   - `web/Makefile`: `frontend` target → `pnpm install --frozen-lockfile && pnpm build`;
     `frontend-test` target → `pnpm test -- --run`.
   - Root `Makefile`: hook `frontend-test` into `ci`; document `make -C web frontend`
     as the regeneration step in root CLAUDE.md's command table.
   - CI (`.github/workflows/ci.yml`): add a node/pnpm setup + `make -C web frontend-test`
     step, and a **staleness guard**: run the build and `git diff --exit-code -- web/internal/web/static/dist`
     so a stale committed bundle fails CI instead of shipping silently.
6. `.gitignore`: `web/frontend/node_modules`.

## Refactor
- Confirm the bundle really is dependency-free (`grep -c "node_modules" dist` → 0;
  vite lib mode with no imports guarantees it, verify anyway).
- Cache header check: `/static/{file}` handler serves with
  `Cache-Control: immutable` (existing `withCacheControl`) — the bundle filename is
  stable, so add a cache-busting query (`?v={{.BuildID}}`) in the page template now,
  while there is exactly one call site. Simplest BuildID: embed the bundle's FNV hash
  computed at server start.

## Verify
- `make ci` green (Go tests + vitest + vet).
- `make -C web frontend && git status` — clean tree after a fresh build (determinism).
- `make up`, open `/locations/sopot/forecast` — placeholder text renders with real
  payload numbers; browser console clean.

## Commit
`feat(web): TypeScript frontend workspace building embedded window-widget bundle`
