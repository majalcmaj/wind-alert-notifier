
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

data "archive_file" "alert-job" {
  type        = "zip"
  source_file = "${path.module}/../bin/wind-alert-job"
  output_path = "${path.module}/../bin/wind-alert-job.zip"
}

resource "aws_lambda_function" "wind-alert-job" {
  function_name    = "wind-alert-job"
  filename         = data.archive_file.alert-job.output_path
  source_code_hash = data.archive_file.alert-job.output_base64sha256
  role             = aws_iam_role.wind_alert.arn


  runtime     = local.lambda_runtime
  handler     = "wind-alert-job.handler"
  memory_size = 128
  timeout     = 10


  environment {
    variables = {
      OPENWEATHER_TOKEN = var.openweather_token
      ICM_METEO_TOKEN   = var.icm_meteo_token
    }
  }
}
