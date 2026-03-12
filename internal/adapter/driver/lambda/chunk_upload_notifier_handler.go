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
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"video_solicitation_microservice/internal/core/dto"
)

type ChunkUploadNotifierHandler struct {
	dynamoClient *dynamodb.Client
	snsClient    *sns.Client
	snsTopicArn  string
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

	snsTopicArn := os.Getenv("SNS_CHUNK_UPLOADED_TOPIC")
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")

	if snsTopicArn == "" || tableName == "" {
		return nil, fmt.Errorf("SNS_CHUNK_UPLOADED_TOPIC and DYNAMODB_TABLE_NAME must be set")
	}

	return &ChunkUploadNotifierHandler{
		dynamoClient: dynamodb.NewFromConfig(awsCfg),
		snsClient:    sns.NewFromConfig(awsCfg),
		snsTopicArn:  snsTopicArn,
		tableName:    tableName,
	}, nil
}

func (h *ChunkUploadNotifierHandler) Handle(ctx context.Context, sqsEvent events.SQSEvent) error {
	log.Printf("Processing %d SQS messages", len(sqsEvent.Records))

	for _, record := range sqsEvent.Records {
		var snsMessage events.SNSEntity
		if err := json.Unmarshal([]byte(record.Body), &snsMessage); err != nil {
			log.Printf("Error parsing SNS message: %v", err)
			continue
		}

		var s3Event events.S3Event
		if err := json.Unmarshal([]byte(snsMessage.Message), &s3Event); err != nil {
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

	_, err = h.snsClient.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(h.snsTopicArn),
		Message:  aws.String(string(messageJSON)),
	})
	if err != nil {
		return fmt.Errorf("failed to publish to SNS: %w", err)
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
	result, err := h.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(h.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: videoID},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}

	metadata := &VideoMetadata{
		UserID:         getStringValue(result.Item, "user_id"),
		UserName:       getStringValue(result.Item, "user_name"),
		UserEmail:      getStringValue(result.Item, "user_email"),
		FramePerSecond: getIntValue(result.Item, "frames_per_second"),
		TotalChunks:    getIntValue(result.Item, "total_chunks"),
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
