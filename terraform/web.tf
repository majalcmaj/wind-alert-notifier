variable "admin_user" {
  description = "Basic-auth username for the wind-alert-web admin UI"
  type        = string
  sensitive   = true
}

variable "admin_password" {
  description = "Basic-auth password for the wind-alert-web admin UI"
  type        = string
  sensitive   = true
}

# IAM

resource "aws_iam_role" "wind_alert_web" {
  name = "wind-alert-web"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "wind_alert_web_basic_execution" {
  role       = aws_iam_role.wind_alert_web.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy" "wind_alert_web_dynamodb" {
  name = "wind-alert-web-dynamodb"
  role = aws_iam_role.wind_alert_web.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "dynamodb:Scan",
        "dynamodb:Query",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:BatchWriteItem",
      ]
      Resource = [
        aws_dynamodb_table.locations.arn,
        aws_dynamodb_table.rules.arn,
      ]
    }]
  })
}

# Lambda

data "archive_file" "web" {
  type        = "zip"
  source_file = "${path.module}/../bin/wind-alert-web/bootstrap"
  output_path = "${path.module}/../bin/wind-alert-web.zip"
}

resource "aws_lambda_function" "wind-alert-web" {
  function_name    = "wind-alert-web"
  filename         = data.archive_file.web.output_path
  source_code_hash = data.archive_file.web.output_base64sha256
  role             = aws_iam_role.wind_alert_web.arn

  runtime     = local.lambda_runtime
  handler     = "handler"
  memory_size = 128
  timeout     = 10

  environment {
    variables = {
      ADMIN_USER     = var.admin_user
      ADMIN_PASSWORD = var.admin_password
    }
  }
}

resource "aws_lambda_function_url" "wind_alert_web" {
  function_name      = aws_lambda_function.wind-alert-web.function_name
  authorization_type = "NONE"
}
