#!/usr/bin/env bash
set -euo pipefail

REGION="${AWS_DEFAULT_REGION:-eu-central-1}"
ENDPOINT_URL=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --local)
      ENDPOINT_URL="http://localhost:8005"
      shift
      ;;
    --endpoint-url)
      ENDPOINT_URL="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

AWS_ARGS="--region $REGION"
if [[ -n "$ENDPOINT_URL" ]]; then
  AWS_ARGS="$AWS_ARGS --endpoint-url $ENDPOINT_URL"
fi

# Create tables if they don't exist (useful for local)
if [[ -n "$ENDPOINT_URL" ]]; then
  echo "Creating tables in local DynamoDB..."
  aws dynamodb create-table $AWS_ARGS --table-name wind-alert-locations \
    --attribute-definitions AttributeName=id,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST || true

  aws dynamodb create-table $AWS_ARGS --table-name wind-alert-rules \
    --attribute-definitions AttributeName=location_id,AttributeType=S AttributeName=name,AttributeType=S \
    --key-schema AttributeName=location_id,KeyType=HASH AttributeName=name,KeyType=RANGE \
    --billing-mode PAY_PER_REQUEST || true
fi

aws dynamodb put-item $AWS_ARGS --table-name wind-alert-locations \
  --item '{"id":{"S":"sopot"},"name":{"S":"Sopot"},"lat":{"N":"54.646034"},"lon":{"N":"18.512407"}}'

aws dynamodb put-item $AWS_ARGS --table-name wind-alert-rules \
  --item '{"location_id":{"S":"sopot"},"name":{"S":"Strong NW afternoon wind"},"speed":{"M":{"from":{"N":"6"},"to":{"N":"25"}}},"angle":{"M":{"from":{"N":"270"},"to":{"N":"360"}}},"hour":{"M":{"from":{"N":"12"},"to":{"N":"20"}}}}'

aws dynamodb put-item $AWS_ARGS --table-name wind-alert-rules \
  --item '{"location_id":{"S":"sopot"},"name":{"S":"Strong W morning wind"},"speed":{"M":{"from":{"N":"5"},"to":{"N":"25"}}},"angle":{"M":{"from":{"N":"247"},"to":{"N":"293"}}},"hour":{"M":{"from":{"N":"8"},"to":{"N":"14"}}}}'

aws dynamodb put-item $AWS_ARGS --table-name wind-alert-rules \
  --item '{"location_id":{"S":"sopot"},"name":{"S":"Any strong wind"},"speed":{"M":{"from":{"N":"8"},"to":{"N":"30"}}},"angle":{"M":{"from":{"N":"0"},"to":{"N":"360"}}},"hour":{"M":{"from":{"N":"6"},"to":{"N":"22"}}}}'

echo "Seeded wind-alert-locations and wind-alert-rules"
