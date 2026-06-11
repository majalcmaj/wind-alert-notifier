# DynamoDB tables — shared by the forecaster (read-only) and web (CRUD) lambdas.

resource "aws_dynamodb_table" "locations" {
  name         = "wind-alert-locations"
  billing_mode = "PAY_PER_REQUEST"

  attribute {
    name = "id"
    type = "S"
  }

  hash_key = "id"
}

resource "aws_dynamodb_table" "rules" {
  name         = "wind-alert-rules"
  billing_mode = "PAY_PER_REQUEST"

  attribute {
    name = "location_id"
    type = "S"
  }

  attribute {
    name = "name"
    type = "S"
  }

  hash_key  = "location_id"
  range_key = "name"
}
