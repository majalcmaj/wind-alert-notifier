#!/usr/bin/env bash
set -euo pipefail

REGION="${AWS_DEFAULT_REGION:-eu-central-1}"

aws dynamodb put-item --region "$REGION" --table-name wind-alert-locations \
  --item '{"id":{"S":"sopot"},"name":{"S":"Sopot"},"lat":{"N":"54.646034"},"lon":{"N":"18.512407"}}'

aws dynamodb put-item --region "$REGION" --table-name wind-alert-rules \
  --item '{"location_id":{"S":"sopot"},"name":{"S":"Strong NW afternoon wind"},"speed":{"M":{"from":{"N":"6"},"to":{"N":"25"}}},"angle":{"M":{"from":{"N":"270"},"to":{"N":"360"}}},"hour":{"M":{"from":{"N":"12"},"to":{"N":"20"}}}}'

aws dynamodb put-item --region "$REGION" --table-name wind-alert-rules \
  --item '{"location_id":{"S":"sopot"},"name":{"S":"Strong W morning wind"},"speed":{"M":{"from":{"N":"5"},"to":{"N":"25"}}},"angle":{"M":{"from":{"N":"247"},"to":{"N":"293"}}},"hour":{"M":{"from":{"N":"8"},"to":{"N":"14"}}}}'

aws dynamodb put-item --region "$REGION" --table-name wind-alert-rules \
  --item '{"location_id":{"S":"sopot"},"name":{"S":"Any strong wind"},"speed":{"M":{"from":{"N":"8"},"to":{"N":"30"}}},"angle":{"M":{"from":{"N":"0"},"to":{"N":"360"}}},"hour":{"M":{"from":{"N":"6"},"to":{"N":"22"}}}}'

echo "Seeded wind-alert-locations and wind-alert-rules in $REGION"
