#!/usr/bin/env bash
set -euo pipefail

ENDPOINT="${AWS_ENDPOINT_URL:-http://localhost:4566}"
REGION="${AWS_REGION:-us-east-1}"
BUCKET="${S3_BUCKET:-applyflow-uploads}"
QUEUE_NAME="${SQS_QUEUE_NAME:-applyflow-email-queue}"

export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION="$REGION"

echo "Creating S3 bucket: $BUCKET"
aws --endpoint-url="$ENDPOINT" s3 mb "s3://$BUCKET" 2>/dev/null || true

echo "Creating SQS queue: $QUEUE_NAME"
QUEUE_URL=$(aws --endpoint-url="$ENDPOINT" sqs create-queue --queue-name "$QUEUE_NAME" --query 'QueueUrl' --output text 2>/dev/null || \
  aws --endpoint-url="$ENDPOINT" sqs get-queue-url --queue-name "$QUEUE_NAME" --query 'QueueUrl' --output text)

echo "SQS_QUEUE_URL=$QUEUE_URL"
echo "Setup complete."
