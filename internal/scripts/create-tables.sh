#!/teusr/bin/env bash
set -euo pipefail

ENDPOINT="${DYNAMODB_ENDPOINT:-http://localhost:8000}"
REGION="${AWS_REGION:-eu-central-1}"

AWS_OPTS="--endpoint-url $ENDPOINT --region $REGION"
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-local}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-local}"

until aws dynamodb list-tables $AWS_OPTS >/dev/null 2>&1; do
  echo "Waiting for DynamoDB Local..."
  sleep 1
done

create_table() {
  aws dynamodb create-table $AWS_OPTS "$@" 2>&1 | grep -v "ResourceInUseException" || true
}

create_table \
  --table-name wind-alert-locations \
  --attribute-definitions AttributeName=id,AttributeType=S \
  --key-schema AttributeName=id,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST

create_table \
  --table-name wind-alert-rules \
  --attribute-definitions \
    AttributeName=location_id,AttributeType=S \
    AttributeName=name,AttributeType=S \
  --key-schema \
    AttributeName=location_id,KeyType=HASH \
    AttributeName=name,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST

echo "Tables ready."
