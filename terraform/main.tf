provider "aws" {
  region = "eu-central-1"
}

locals {
  account_id = "513532022998"
  region     = "eu-central-1"
  ecr_image  = "${local.account_id}.dkr.ecr.${local.region}.amazonaws.com/wind-alert:latest"
}

variable "openweather_token" {
  description = "OpenWeather API token"
  type        = string
  sensitive   = true
}

variable "icm_meteo_token" {
  description = "ICM Meteo (api.meteo.pl) API token"
  type        = string
  sensitive   = true
}

# ECR

resource "aws_ecr_repository" "wind_alert" {
  name = "wind-alert"

  image_scanning_configuration {
    scan_on_push = true
  }
}

# DynamoDB

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

# IAM

resource "aws_iam_role" "wind_alert" {
  name = "wind-alert"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "basic_execution" {
  role       = aws_iam_role.wind_alert.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy" "dynamodb" {
  name = "wind-alert-dynamodb"
  role = aws_iam_role.wind_alert.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = ["dynamodb:Scan", "dynamodb:Query"]
      Resource = [
        aws_dynamodb_table.locations.arn,
        aws_dynamodb_table.rules.arn,
      ]
    }]
  })
}

resource "aws_iam_role_policy" "ses" {
  name = "wind-alert-ses"
  role = aws_iam_role.wind_alert.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["ses:SendEmail", "sesv2:SendEmail"]
      Resource = "*"
    }]
  })
}

# Lambda

resource "aws_lambda_function" "wind_alert" {
  function_name = "wind-alert"
  role          = aws_iam_role.wind_alert.arn
  package_type  = "Image"
  image_uri     = local.ecr_image
  memory_size   = 128
  timeout       = 10

  environment {
    variables = {
      OPENWEATHER_TOKEN = var.openweather_token
      ICM_METEO_TOKEN   = var.icm_meteo_token
    }
  }

  # image_uri is managed by CI (aws lambda update-function-code after each push)
  lifecycle {
    ignore_changes = [image_uri]
  }
}
