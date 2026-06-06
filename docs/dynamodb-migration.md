# Plan: Move Rules and Locations to DynamoDB

## Context

Rules and locations are currently embedded JSON blobs compiled into the Lambda binary (`internal/rules.json`, `internal/locations.json`). Moving them to DynamoDB enables runtime edits without redeployment. DynamoDB is the right choice: `aws-sdk-go-v2` is already in the project (SES uses it), there's no VPC/connection-pooling overhead, and the cost at ~1500 reads/month is fractions of a cent.

---

## Data Flow (before → after)

```
BEFORE                              AFTER
------                              -----
init() reads rules.json    →        Lambda calls DynamoDB on each invocation
init() reads locations.json →       LoadLocations (Scan) + LoadRulesForLocation (Query)
main.go iterates globals   →        main.go iterates returned slices
```

```
main.go handler
  │
  ├─ newDynamoDBClient(ctx)
  │
  ├─ LoadLocations(ctx, client)          ← Scan wind-alert-locations
  │    returns []Location
  │
  └─ for each loc:
       LoadRulesForLocation(ctx, client, loc.ID)   ← Query wind-alert-rules PK=loc.ID
         returns []Rule
       → EvaluateForecast(forecast, locRules)  [unchanged]
```

---

## DynamoDB Table Design

**`wind-alert-locations`** — hash key: `id` (S)
| id | name | lat | lon |
|----|------|-----|-----|
| sopot | Sopot | 54.646034 | 18.512407 |

**`wind-alert-rules`** — hash key: `location_id` (S), sort key: `name` (S)

All other rule fields stored as nested DynamoDB Maps (`angle`, `speed`, `hour` each with `from`/`to`).

Rules **must** have a `Name` (it is the sort key). All three existing rules already have names.

---

## Code Changes

### 1. Struct tags — `internal/rule_engine.go`, `internal/locations.go`

Add `dynamodbav` tags to enable `attributevalue.UnmarshalListOfMaps` / `attributevalue.UnmarshalMap` — purely additive, no logic change, no test breakage.

`Range` (in `rule_engine.go`):
```go
type Range struct {
    From float64 `json:"from" dynamodbav:"from"`
    To   float64 `json:"to"   dynamodbav:"to"`
}
```

`Rule` (in `rule_engine.go`):
```go
type Rule struct {
    Name       string `json:"name,omitempty"  dynamodbav:"name,omitempty"`
    LocationID string `json:"location_id"     dynamodbav:"location_id"`
    AngleRange Range  `json:"angle"           dynamodbav:"angle"`
    SpeedRange Range  `json:"speed"           dynamodbav:"speed"`
    HourRange  Range  `json:"hour"            dynamodbav:"hour"`
}
```

`Location` (in `locations.go`):
```go
type Location struct {
    ID   string  `json:"id"   dynamodbav:"id"`
    Name string  `json:"name" dynamodbav:"name"`
    Lat  float64 `json:"lat"  dynamodbav:"lat"`
    Lon  float64 `json:"lon"  dynamodbav:"lon"`
}
```

### 2. `internal/locations.go` — remove embed machinery

Keep only the `Location` struct (with new tags). Delete: `import "embed"`, `//go:embed`, `var locationsJSON`, `var Locations`, `func init()`.

### 3. `internal/rules.go` — delete file

The file only contains the embed + global var. `Rule` struct lives in `rule_engine.go`. Delete `rules.go` and `rules.json`. Also delete `locations.json`.

### 4. `internal/dynamodb.go` — new file

```go
package internal

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    ddbexpr "github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
)

func LoadLocations(ctx context.Context, client *dynamodb.Client) ([]Location, error) {
    out, err := client.Scan(ctx, &dynamodb.ScanInput{TableName: aws.String("wind-alert-locations")})
    if err != nil {
        return nil, err
    }
    var locs []Location
    return locs, attributevalue.UnmarshalListOfMaps(out.Items, &locs)
}

func LoadRulesForLocation(ctx context.Context, client *dynamodb.Client, locationID string) ([]Rule, error) {
    expr, _ := ddbexpr.NewBuilder().
        WithKeyCondition(ddbexpr.Key("location_id").Equal(ddbexpr.Value(locationID))).
        Build()
    out, err := client.Query(ctx, &dynamodb.QueryInput{
        TableName:                 aws.String("wind-alert-rules"),
        KeyConditionExpression:    expr.KeyCondition(),
        ExpressionAttributeNames:  expr.Names(),
        ExpressionAttributeValues: expr.Values(),
    })
    if err != nil {
        return nil, err
    }
    var rules []Rule
    return rules, attributevalue.UnmarshalListOfMaps(out.Items, &rules)
}
```

(Note: exact import paths and `aws.String` usage follow existing project patterns from `main.go`/SES code.)

### 5. `main.go` — replace globals with DynamoDB calls

Add a `newDynamoDBClient(ctx)` function parallel to `newMailSender`, returning `*dynamodb.Client`. In `handler`:

```go
dynamoClient, err := newDynamoDBClient(ctx)
if err != nil { return nil, err }

locations, err := internal.LoadLocations(ctx, dynamoClient)
if err != nil { return nil, err }

for _, loc := range locations {
    locRules, err := internal.LoadRulesForLocation(ctx, dynamoClient, loc.ID)
    if err != nil { return nil, err }
    // rest of loop unchanged — forecast, EvaluateForecast, etc.
}
```

Remove references to `internal.Locations` and `internal.AlertRules`. The inner `r.LocationID == loc.ID` filter loop is removed (DynamoDB Query already scopes by location).

`newDynamoDBClient` returns `*dynamodb.Client` directly (no LOCAL_MODE branch needed — DynamoDB is accessed the same way in all environments using real AWS credentials).

### 6. `go.mod` — add DynamoDB dependencies

Run `go get github.com/aws/aws-sdk-go-v2/service/dynamodb` and `go get github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue`. These pull in the same SDK infrastructure already present for SES.

### 7. `aws/template.yml` — add tables + IAM

Add two `AWS::DynamoDB::Table` resources (PAY_PER_REQUEST) and add to the Lambda's `Policies` statement:

```yaml
WindAlertLocationsTable:
  Type: AWS::DynamoDB::Table
  Properties:
    TableName: wind-alert-locations
    BillingMode: PAY_PER_REQUEST
    AttributeDefinitions:
      - { AttributeName: id, AttributeType: S }
    KeySchema:
      - { AttributeName: id, KeyType: HASH }

WindAlertRulesTable:
  Type: AWS::DynamoDB::Table
  Properties:
    TableName: wind-alert-rules
    BillingMode: PAY_PER_REQUEST
    AttributeDefinitions:
      - { AttributeName: location_id, AttributeType: S }
      - { AttributeName: name,        AttributeType: S }
    KeySchema:
      - { AttributeName: location_id, KeyType: HASH }
      - { AttributeName: name,        KeyType: RANGE }
```

IAM addition to the `WindalertGo` function policies:
```yaml
- Effect: Allow
  Action:
    - dynamodb:Scan
    - dynamodb:Query
  Resource:
    - !Sub arn:aws:dynamodb:${AWS::Region}:${AWS::AccountId}:table/wind-alert-locations
    - !Sub arn:aws:dynamodb:${AWS::Region}:${AWS::AccountId}:table/wind-alert-rules
```

### 8. Seed script — `scripts/seed-dynamodb.sh`

One-time shell script using AWS CLI to seed the existing data:

```bash
#!/usr/bin/env bash
REGION=eu-central-1

aws dynamodb put-item --region $REGION --table-name wind-alert-locations \
  --item '{"id":{"S":"sopot"},"name":{"S":"Sopot"},"lat":{"N":"54.646034"},"lon":{"N":"18.512407"}}'

aws dynamodb put-item --region $REGION --table-name wind-alert-rules \
  --item '{"location_id":{"S":"sopot"},"name":{"S":"Strong NW afternoon wind"},"speed":{"M":{"from":{"N":"6"},"to":{"N":"25"}}},"angle":{"M":{"from":{"N":"270"},"to":{"N":"360"}}},"hour":{"M":{"from":{"N":"12"},"to":{"N":"20"}}}}'
# ... (2 more rules)
```

---

## Files Summary

| File | Action |
|------|--------|
| `internal/rule_engine.go` | Add `dynamodbav` tags to `Range`, `Rule` |
| `internal/locations.go` | Add `dynamodbav` tags; remove embed/init/global |
| `internal/rules.go` | **Delete** |
| `internal/locations.json` | **Delete** |
| `internal/rules.json` | **Delete** |
| `internal/dynamodb.go` | **New** — `LoadLocations`, `LoadRulesForLocation` |
| `main.go` | Add DynamoDB client; replace globals with loader calls |
| `aws/template.yml` | Add DynamoDB table resources + IAM |
| `go.mod` / `go.sum` | Add dynamodb + attributevalue packages |
| `scripts/seed-dynamodb.sh` | **New** — one-time seed script |

---

## Verification

1. `make test` — all existing unit tests pass unchanged (they construct structs directly, none reference `internal.Locations` / `internal.AlertRules`)
2. `make build` — compiles cleanly
3. Deploy tables via SAM (`sam deploy`) or create manually in AWS console / CLI
4. Run `scripts/seed-dynamodb.sh` to populate tables
5. `make run-docker && make run-test-request` — Lambda reads from real DynamoDB, mail rendered to stdout
