#!/usr/bin/env bash
set -e

echo "🚀 Starting LocalStack bootstrap..."

# =====================
# S3 Buckets
# =====================
BUCKET_NAME=video-solicitation-bucket

echo "▶ Creating S3 bucket '$BUCKET_NAME'..."

awslocal s3 mb s3://$BUCKET_NAME

awslocal s3api put-bucket-cors \
    --bucket $BUCKET_NAME \
    --cors-configuration '{
  "CORSRules": [
    {
      "AllowedHeaders": ["*"],
      "AllowedMethods": ["PUT", "POST", "GET", "HEAD"],
      "AllowedOrigins": ["*"],
      "ExposeHeaders": ["ETag"]
    }
  ]
}'

echo "✅ S3 bucket '$BUCKET_NAME' created"


# =====================
# SNS TOPICS
# =====================
ALL_CHUNK_PROCESSED_EVENT=all-chunk-processed
VIDEO_PROCESSED_ERROR_EVENT=video-processing-error

echo "▶ Creating SNS topics..."

ALL_CHUNK_PROCESSED_EVENT_ARN=$(awslocal sns create-topic \
  --name "$ALL_CHUNK_PROCESSED_EVENT" \
  --query 'TopicArn' \
  --output text)

echo "✅ SNS topic '$ALL_CHUNK_PROCESSED_EVENT' created"

VIDEO_PROCESSED_ERROR_EVENT_ARN=$(awslocal sns create-topic \
  --name "$VIDEO_PROCESSED_ERROR_EVENT" \
  --query 'TopicArn' \
  --output text)

echo "✅ SNS topic '$VIDEO_PROCESSED_ERROR_EVENT' created"



# =====================
# SQS QUEUES
# =====================
echo "▶ Creating SQS queues..."

UPDATE_VIDEO_CHUNK_STATUS_QUEUE=update-video-chunk-status
UPDATE_VIDEO_STATUS_QUEUE=update-video-status
VIDEO_PROCESSING_ERROR_QUEUE=video-processing-error

UPDATE_VIDEO_CHUNK_STATUS_QUEUE_URL=$(awslocal sqs create-queue \
    --queue-name $UPDATE_VIDEO_CHUNK_STATUS_QUEUE \
    --query 'QueueUrl' \
    --output text)

echo "✅ SQS queue '$UPDATE_VIDEO_CHUNK_STATUS_QUEUE' created"

UPDATE_VIDEO_STATUS_QUEUE_URL=$(awslocal sqs create-queue \
    --queue-name $UPDATE_VIDEO_STATUS_QUEUE \
    --query 'QueueUrl' \
    --output text)

echo "✅ SQS queue '$UPDATE_VIDEO_STATUS_QUEUE' created"

VIDEO_PROCESSING_ERROR_QUEUE_URL=$(awslocal sqs create-queue \
    --queue-name $VIDEO_PROCESSING_ERROR_QUEUE \
    --query 'QueueUrl' \
    --output text)

echo "✅ SQS queue '$VIDEO_PROCESSING_ERROR_QUEUE' created"

echo "▶ Getting Queue ARNs..."

UPDATE_VIDEO_CHUNK_STATUS_QUEUE_ARN=$(awslocal sqs get-queue-attributes \
  --queue-url "$UPDATE_VIDEO_CHUNK_STATUS_QUEUE_URL" \
  --attribute-names QueueArn \
  --query 'Attributes.QueueArn' \
  --output text)

UPDATE_VIDEO_STATUS_QUEUE_ARN=$(awslocal sqs get-queue-attributes \
  --queue-url "$UPDATE_VIDEO_STATUS_QUEUE_URL" \
  --attribute-names QueueArn \
  --query 'Attributes.QueueArn' \
  --output text)

VIDEO_PROCESSING_ERROR_QUEUE_ARN=$(awslocal sqs get-queue-attributes \
  --queue-url "$VIDEO_PROCESSING_ERROR_QUEUE_URL" \
  --attribute-names QueueArn \
  --query 'Attributes.QueueArn' \
  --output text)

echo "✅ SQS queue ARNs retrieved"

echo "▶ Subscribing SQS queues to SNS topics..."

awslocal sns subscribe \
  --topic-arn "$ALL_CHUNK_PROCESSED_EVENT_ARN" \
  --protocol sqs \
  --notification-endpoint "$UPDATE_VIDEO_CHUNK_STATUS_QUEUE_ARN"

awslocal sns subscribe \
  --topic-arn "$VIDEO_PROCESSED_ERROR_EVENT_ARN" \
  --protocol sqs \
  --notification-endpoint "$VIDEO_PROCESSING_ERROR_QUEUE_ARN"

echo "✅ SQS queues subscribed to SNS topics"

echo "✅ LocalStack bootstrap finished successfully"
