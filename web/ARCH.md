# Wind Alert Web — Architecture

Admin frontend + config CRUD backend for [`../alert-job`](../alert-job). Lets an operator
manage **locations** and **alert rules** stored in the shared DynamoDB tables.
Scope: configuration only. Alert/forecast data hosting is future work, not covered here.

## 1. System context

```
Browser (htmx + vanilla JS)
        │  HTTPS, Basic-auth
        ▼
Lambda Function URL  ──►  wind-alert-web Lambda (Go, Docker, single function)
                                    │  PutItem / UpdateItem / DeleteItem / Scan / Query
                                    ▼
                          DynamoDB  (provisioned by ../terraform)
                          ├─ wind-alert-locations
                          └─ wind-alert-rules
                                    ▲
                                    │  Scan / Query (read-only)
                          ../alert-job Lambda (forecast evaluator, separate function)
```

Two independent Lambdas share the two tables. `../alert-job` reads config and sends alerts;
`wind-alert-web` is the **only writer**. Both lambdas are deployed from this monorepo and share
the `Location`/`Rule`/`Range` schema and DynamoDB access code via the `../shared` Go module —
the tables themselves are provisioned once in `../terraform`, not owned by either lambda.

## 2. Components

### 2.1 Single lambdalith (Go)
One Lambda behind a **Lambda Function URL**. It is the whole backend:
- Serves full HTML pages (initial load) and **htmx HTML fragments** (interactions).
- Performs all DynamoDB CRUD.
- Embeds static assets (htmx lib, CSS) in the binary via `//go:embed` — no S3/CloudFront needed.

Rationale: admin traffic is tiny, htmx implies many small routes best handled by in-process
routing, and it reuses the existing Docker + SAM pipeline 1:1. No benefit to splitting functions.

**Routing:** Go 1.22+ `net/http.ServeMux` method+path patterns. The Function URL payload
(API-GW-v2 / payload format 2.0 shape) is adapted to `http.Handler` via
`github.com/awslabs/aws-lambda-go-api-proxy/httpadapter` (Function URL adapter). Handlers are
plain `net/http` — testable with `httptest` without any AWS in the loop.

### 2.2 Frontend (htmx + vanilla JS, server-rendered)
- Server renders fragments; htmx swaps them in. Minimal vanilla JS only for widgets that need it
  (e.g. compass/angle picker, map coordinate pick — optional, later).
- `html/template` templates embedded in the binary. One layout + per-resource partials
  (list row, edit form).
- Styling: a small classless/utility CSS (e.g. Pico.css) vendored + embedded. No build step for CSS.
- htmx library vendored + embedded (served from the Lambda), so the app has **no external CDN dependency**.

### 2.3 Data access layer
DynamoDB CRUD lives in the shared `../shared/dynamo` package (`dynamo.Store`): `PutLocation`,
`DeleteLocation`, `PutRule`, `DeleteRule`, `DeleteRulesForLocation`, plus the `LoadLocations` /
`LoadRulesForLocation` read patterns also used by `../alert-job`. Uses `aws-sdk-go-v2` +
`feature/dynamodb/attributevalue` and `expression`. `main.go` constructs one `dynamo.Store` via
`dynamo.New(ctx)` and passes it to the server as the `Datastore`.

### 2.4 Shared schema
`Location`, `Rule`, `Range` structs and their `dynamodbav` tags live in `../shared/model`,
imported by both lambdas — items are byte-compatible by construction, not by convention. Form
validation specific to this admin UI (`ValidateLocation`, `ValidateRule`) stays local, in
`internal/validate`.

### 2.5 Auth
Basic-auth middleware wrapping all routes. Credentials from env vars
(`ADMIN_USER`, `ADMIN_PASSWORD`), injected via SAM template parameters / Lambda env. Function URL
auth type `NONE` (basic-auth is enforced in-app). Swap-in target later: CloudFront + Cognito.

## 3. Data model & validation

Unchanged from backend. Server-side validation before any write:
- **Location:** `id` non-empty slug; `name` non-empty; `lat ∈ [-90,90]`; `lon ∈ [-180,180]`.
- **Rule:** `location_id` must reference an existing location; `name` non-empty (it's the SK);
  `speed.from ≤ speed.to`, `speed ≥ 0`; `angle ∈ [0,360]`; `hour ∈ [0,24]`;
  `min_confidence ∈ [0,1]`.
- **Cyclic ranges are valid:** `from > to` is allowed for `angle` (wrap past 360°) and `hour`
  (overnight). Do **not** reject these. UI should hint the wrap behavior.

## 4. HTTP surface (htmx routes)

Method+path patterns; responses are HTML (full page on `GET /`, fragments elsewhere).

```
GET    /                              → page: locations list
GET    /static/{file}                 → embedded htmx / css

GET    /locations                     → fragment: locations list
GET    /locations/new                 → fragment: create form
POST   /locations                     → create, return updated list/row
GET    /locations/{id}/edit           → fragment: edit form
PUT    /locations/{id}                → update, return row
DELETE /locations/{id}                → delete (also cascade-delete its rules), return empty

GET    /locations/{id}/rules          → fragment: rules for a location
GET    /locations/{id}/rules/new      → fragment: create rule form
POST   /locations/{id}/rules          → create rule
GET    /locations/{id}/rules/{name}/edit → fragment: edit rule form
PUT    /locations/{id}/rules/{name}   → update rule
DELETE /locations/{id}/rules/{name}   → delete rule
```

Rules are nested under their location (matches the PK/SK shape). htmx uses
`hx-get/post/put/delete` + `hx-target`/`hx-swap`; Function URL passes real HTTP methods through,
so no `_method` override is needed.

> Note: editing a rule's `name` rewrites the SK → implement update as delete-old + put-new when
> `name` changes; otherwise plain `PutItem`.

## 5. Infrastructure (Terraform, `../terraform`)

Provisioned alongside `../alert-job` in the shared Terraform stack (`web.tf`):

- A **Function URL**: `authorization_type = "NONE"` (basic-auth enforced in-app).
- A **DynamoDB IAM policy** scoped to the two table ARNs, actions:
  `Scan, Query, PutItem, UpdateItem, DeleteItem, BatchWriteItem`. The tables themselves are
  defined once in `dynamodb.tf`, shared with `../alert-job`.
- Env vars: `ADMIN_USER`, `ADMIN_PASSWORD` (sensitive Terraform variables).
- `Timeout` ~10s, `MemorySize` 256 MB (DynamoDB round-trips + template render).
- Region/stack: `eu-central-1`. Own ECR repo `wind-alert-web`; GH Actions builds/pushes/deploys
  this lambda independently of `../alert-job`.

## 6. Project layout (target)

```
web/
├── main.go                 # lambda.Start(httpadapter → mux)
├── internal/
│   ├── server/             # router, handlers, basic-auth middleware
│   ├── validate/           # ValidateLocation, ValidateRule (admin-form validation)
│   └── web/
│       ├── templates/      # *.html (embed)
│       └── static/         # htmx.min.js, pico.css (embed)
└── Makefile                # existing build/test; drop stale mjml `generate` target

../shared/
├── model/                   # Location, Rule, Range (dynamodbav-tagged) + Describe
└── dynamo/                  # Store: LoadLocations, PutLocation, LoadRulesForLocation, ...
```

## 7. Out of scope (noted, not designed)
- Hosting alert/forecast data (future).
- Multi-tenancy / per-user config (single operator for now).
- Cognito / hosted login (basic-auth placeholder until then).
- Map/coordinate picker & compass widget (nice-to-have, can land after CRUD works).
