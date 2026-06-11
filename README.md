# Wind Alert

Go multi-module monorepo for two AWS Lambda functions that share a DynamoDB-backed
configuration schema.

## Layout

```
alerting-job/
├── go.work          # workspace spanning shared, alert-job, web
├── shared/          # github.com/majalcmaj/wind-alert/shared
│   ├── model/        Location, Rule, Range types + presentation helpers
│   └── dynamo/       DynamoDB access layer (Store: load/put/delete)
├── alert-job/      # github.com/majalcmaj/wind-alert/alert-job — see alert-job/CLAUDE.md
│   Fetches wind forecasts, evaluates alert rules, emails via SES.
├── web/             # github.com/majalcmaj/wind-alert/web — see web/ARCH.md
│   htmx admin UI + CRUD API for locations and alert rules.
└── terraform/       # infrastructure for both lambdas + shared DynamoDB tables
    ├── dynamodb.tf   wind-alert-locations, wind-alert-rules
    ├── alert-job.tf ECR repo, IAM role, Lambda for alert-job
    └── web.tf        ECR repo, IAM role, Lambda + Function URL for web
```

`alert-job` is read-only against the shared tables; `web` is the only writer. Both lambdas are
built and deployed independently — see `.github/workflows/ci.yml`.

## Building & testing

```bash
go build ./shared/... ./alert-job/... ./web/...
go vet   ./shared/... ./alert-job/... ./web/...
go test  ./shared/... ./alert-job/... ./web/...
```

(`./...` from the repo root doesn't work — there's no `go.mod` here, only `go.work`.)

For module-specific commands (Docker builds, lint, local run), see `alert-job/CLAUDE.md` and
`web/ARCH.md`.
