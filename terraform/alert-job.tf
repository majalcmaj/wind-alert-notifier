
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

resource "aws_lambda_function" "wind_alert" {
  function_name = "wind-alert"
  role          = aws_iam_role.wind_alert.arn
  package_type  = "Image"
  image_uri     = "${local.ecr_url}/wind-alert:latest" 
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
