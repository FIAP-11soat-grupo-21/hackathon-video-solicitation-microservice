package lambda

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"video_solicitation_microservice/internal/core/dto"
)

type ChunkUploadNotifierHandler struct {
	dynamoClient *dynamodb.Client
	sqsClient    *sqs.Client
	sqsQueueUrl  string
	tableName    string
}

func NewChunkUploadNotifierHandler() (*ChunkUploadNotifierHandler, error) {
	ctx := context.Background()
	
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-2"
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	sqsQueueUrl := os.Getenv("SQS_CHUNK_PROCESSOR_QUEUE_URL")
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")

	if sqsQueueUrl == "" || tableName == "" {
		return nil, fmt.Errorf("SQS_CHUNK_PROCESSOR_QUEUE_URL and DYNAMODB_TABLE_NAME must be set")
	}

	return &ChunkUploadNotifierHandler{
		dynamoClient: dynamodb.NewFromConfig(awsCfg),
		sqsClient:    sqs.NewFromConfig(awsCfg),
		sqsQueueUrl:  sqsQueueUrl,
		tableName:    tableName,
	}, nil
}

func (h *ChunkUploadNotifierHandler) Handle(ctx context.Context, snsEvent events.SNSEvent) error {
	log.Printf("Processing %d SNS messages", len(snsEvent.Records))

	for _, record := range snsEvent.Records {
		var s3Event events.S3Event
		if err := json.Unmarshal([]byte(record.SNS.Message), &s3Event); err != nil {
			log.Printf("Error parsing S3 event: %v", err)
			continue
		}

		for _, s3Record := range s3Event.Records {
			if err := h.processS3Event(ctx, s3Record); err != nil {
				log.Printf("Error processing S3 event: %v", err)
				return err
			}
		}
	}

	return nil
}

func (h *ChunkUploadNotifierHandler) processS3Event(ctx context.Context, s3Record events.S3EventRecord) error {
	bucket := s3Record.S3.Bucket.Name
	key := s3Record.S3.Object.Key

	log.Printf("Processing S3 event: bucket=%s, key=%s", bucket, key)

	videoID, chunkPart, err := parseS3Key(key)
	if err != nil {
		return fmt.Errorf("failed to parse S3 key: %w", err)
	}

	log.Printf("Extracted: video_id=%s, chunk_part=%d", videoID, chunkPart)

	metadata, err := h.getVideoMetadata(ctx, videoID)
	if err != nil {
		return fmt.Errorf("failed to get video metadata: %w", err)
	}

	message := dto.ChunkUploadedDTO{
		Bucket:        bucket,
		VideoObjectID: key,
		User: dto.UserDTO{
			ID:    metadata.UserID,
			Name:  metadata.UserName,
			Email: metadata.UserEmail,
		},
		ImageDestination: fmt.Sprintf("videos/%s/images", videoID),
		FramePerSecond:   metadata.FramePerSecond,
		ChunkPart:        chunkPart,
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = h.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(h.sqsQueueUrl),
		MessageBody: aws.String(string(messageJSON)),
	})
	if err != nil {
		return fmt.Errorf("failed to send message to SQS: %w", err)
	}

	log.Printf("Successfully published chunk upload notification for video %s, chunk %d", videoID, chunkPart)
	return nil
}

func parseS3Key(key string) (videoID string, chunkPart int, err error) {
	re := regexp.MustCompile(`videos/([^/]+)/chunks/part_(\d+)\.mp4`)
	matches := re.FindStringSubmatch(key)

	if len(matches) != 3 {
		return "", 0, fmt.Errorf("invalid S3 key format: %s", key)
	}

	videoID = matches[1]
	chunkPart, err = strconv.Atoi(matches[2])
	if err != nil {
		return "", 0, fmt.Errorf("invalid chunk number: %w", err)
	}

	return videoID, chunkPart, nil
}

type VideoMetadata struct {
	UserID         string
	UserName       string
	UserEmail      string
	FramePerSecond int
	TotalChunks    int
}

func (h *ChunkUploadNotifierHandler) getVideoMetadata(ctx context.Context, videoID string) (*VideoMetadata, error) {
	result, err := h.dynamoClient.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(h.tableName),
		FilterExpression: aws.String("video_id = :id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id": &types.AttributeValueMemberS{Value: videoID},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}

	item := result.Items[0]
	metadata := &VideoMetadata{
		UserID:         getStringValue(item, "user_id"),
		UserName:       getStringValue(item, "user_name"),
		UserEmail:      getStringValue(item, "user_email"),
		FramePerSecond: getIntValue(item, "frames_per_second"),
		TotalChunks:    getIntValue(item, "total_chunks"),
	}

	return metadata, nil
}

func getStringValue(item map[string]types.AttributeValue, key string) string {
	if val, ok := item[key]; ok {
		if s, ok := val.(*types.AttributeValueMemberS); ok {
			return s.Value
		}
	}
	return ""
}

func getIntValue(item map[string]types.AttributeValue, key string) int {
	if val, ok := item[key]; ok {
		if n, ok := val.(*types.AttributeValueMemberN); ok {
			var result int
			fmt.Sscanf(n.Value, "%d", &result)
			return result
		}
	}
	return 2 // valor padrão: 1 frame a cada 2 segundos
}
