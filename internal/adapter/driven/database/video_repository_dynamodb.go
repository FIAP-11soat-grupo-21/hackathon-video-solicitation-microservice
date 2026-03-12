package database

import (
	context "context"
	"fmt"
	"log"
	"strconv"
	"time"
	"video_solicitation_microservice/internal/core/domain/entity"
	"video_solicitation_microservice/internal/core/domain/port"
	"video_solicitation_microservice/internal/core/domain/value_object"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	defaultVideosTableName = "videos"
	defaultChunksTableName = "chunks"
	batchWriteMaxItems     = 25 // DynamoDB BatchWriteItem limit
)

// Repositório DynamoDB para vídeos

type videoRepositoryDynamoDB struct {
	db              *dynamodb.Client
	videosTableName string
	chunksTableName string
}

func NewVideoRepositoryDynamoDB(db *dynamodb.Client, opts ...func(*videoRepositoryDynamoDB)) port.VideoRepository {
	repo := &videoRepositoryDynamoDB{
		db:              db,
		videosTableName: defaultVideosTableName,
		chunksTableName: defaultChunksTableName,
	}
	for _, opt := range opts {
		opt(repo)
	}
	return repo
}

func WithTableNames(videosTable, chunksTable string) func(*videoRepositoryDynamoDB) {
	return func(r *videoRepositoryDynamoDB) {
		if videosTable != "" {
			r.videosTableName = videosTable
		}
		if chunksTable != "" {
			r.chunksTableName = chunksTable
		}
	}
}

func (r *videoRepositoryDynamoDB) Save(ctx context.Context, video *entity.Video) error {
	item := map[string]types.AttributeValue{
		"video_id":           &types.AttributeValueMemberS{Value: video.ID},
		"user_id":            &types.AttributeValueMemberS{Value: video.User.ID},
		"user_name":          &types.AttributeValueMemberS{Value: video.User.Name},
		"user_email":         &types.AttributeValueMemberS{Value: video.User.Email},
		"file_name":          &types.AttributeValueMemberS{Value: video.Metadata.FileName},
		"duration_seconds":   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", video.Metadata.DurationSeconds)},
		"size_bytes":         &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", video.Metadata.SizeBytes)},
		"status":             &types.AttributeValueMemberS{Value: string(video.Status)},
		"bucket_name":        &types.AttributeValueMemberS{Value: video.FileLocation.BucketName},
		"video_chunk_folder": &types.AttributeValueMemberS{Value: video.FileLocation.VideoChunkFolder},
		"image_folder":       &types.AttributeValueMemberS{Value: video.FileLocation.ImageFolder},
		"download_url":       &types.AttributeValueMemberS{Value: video.FileLocation.DownloadURL},
		"error_cause":        &types.AttributeValueMemberS{Value: video.ErrorCause},
		"created_at":         &types.AttributeValueMemberS{Value: video.CreatedAt.Format(time.RFC3339)},
		"updated_at":         &types.AttributeValueMemberS{Value: video.UpdatedAt.Format(time.RFC3339)},
	}

	_, err := r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.videosTableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to put video item: %w", err)
	}

	// Salvar chunks na tabela chunks
	for _, chunk := range video.Chunks {
		chunkItem := map[string]types.AttributeValue{
			"video_id":         &types.AttributeValueMemberS{Value: video.ID},
			"part_number":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", chunk.PartNumber)},
			"start_time":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", chunk.StartTime)},
			"end_time":         &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", chunk.EndTime)},
			"frame_per_second": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", chunk.FramePerSecond)},
			"status":           &types.AttributeValueMemberS{Value: string(chunk.Status)},
			"video_object_id":  &types.AttributeValueMemberS{Value: chunk.VideoObjectID},
		}
		_, err := r.db.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(r.chunksTableName),
			Item:      chunkItem,
		})
		if err != nil {
			return fmt.Errorf("failed to put chunk item: %w", err)
		}
	}

	return nil
}

func (r *videoRepositoryDynamoDB) FindByID(ctx context.Context, videoID string) (*entity.Video, error) {
	// Buscar vídeo pelo GSI video_id-index
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(r.videosTableName),
		IndexName:              aws.String("video_id-index"),
		KeyConditionExpression: aws.String("video_id = :vid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":vid": &types.AttributeValueMemberS{Value: videoID},
		},
	}
	resp, err := r.db.Query(ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to query video: %w", err)
	}
	if len(resp.Items) == 0 {
		return nil, nil
	}
	item := resp.Items[0]
	video := &entity.Video{
		ID: getString(item, "video_id"),
		User: entity.User{
			ID:    getString(item, "user_id"),
			Name:  getString(item, "user_name"),
			Email: getString(item, "user_email"),
		},
		Metadata: entity.Metadata{
			FileName:        getString(item, "file_name"),
			DurationSeconds: getInt(item, "duration_seconds"),
			SizeBytes:       int64(getInt(item, "size_bytes")),
		},
		Status: value_object.VideoStatus(getString(item, "status")),
		FileLocation: entity.FileLocation{
			BucketName:       getString(item, "bucket_name"),
			VideoChunkFolder: getString(item, "video_chunk_folder"),
			ImageFolder:      getString(item, "image_folder"),
			DownloadURL:      getString(item, "download_url"),
		},
		ErrorCause: getString(item, "error_cause"),
	}
	if t, err := time.Parse(time.RFC3339, getString(item, "created_at")); err == nil {
		video.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, getString(item, "updated_at")); err == nil {
		video.UpdatedAt = t
	}

	// Buscar chunks
	chunkQuery := &dynamodb.QueryInput{
		TableName:              aws.String(r.chunksTableName),
		KeyConditionExpression: aws.String("video_id = :vid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":vid": &types.AttributeValueMemberS{Value: videoID},
		},
	}
	chunkResp, err := r.db.Query(ctx, chunkQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	for _, chunkItem := range chunkResp.Items {
		chunk := entity.Chunk{
			PartNumber:     getInt(chunkItem, "part_number"),
			StartTime:      getInt(chunkItem, "start_time"),
			EndTime:        getInt(chunkItem, "end_time"),
			FramePerSecond: getInt(chunkItem, "frame_per_second"),
			Status:         value_object.ChunkStatus(getString(chunkItem, "status")),
			VideoObjectID:  getString(chunkItem, "video_object_id"),
		}
		video.Chunks = append(video.Chunks, chunk)
	}

	return video, nil
}

func (r *videoRepositoryDynamoDB) FindByUserID(ctx context.Context, userID string) ([]*entity.Video, error) {
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(r.videosTableName),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
	}
	resp, err := r.db.Query(ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to query videos by user_id: %w", err)
	}
	videos := []*entity.Video{}
	for _, item := range resp.Items {
		video := &entity.Video{
			ID: getString(item, "video_id"),
			User: entity.User{
				ID:    getString(item, "user_id"),
				Name:  getString(item, "user_name"),
				Email: getString(item, "user_email"),
			},
			Metadata: entity.Metadata{
				FileName:        getString(item, "file_name"),
				DurationSeconds: getInt(item, "duration_seconds"),
				SizeBytes:       int64(getInt(item, "size_bytes")),
			},
			Status: value_object.VideoStatus(getString(item, "status")),
			FileLocation: entity.FileLocation{
				BucketName:       getString(item, "bucket_name"),
				VideoChunkFolder: getString(item, "video_chunk_folder"),
				ImageFolder:      getString(item, "image_folder"),
				DownloadURL:      getString(item, "download_url"),
			},
			ErrorCause: getString(item, "error_cause"),
		}
		if t, err := time.Parse(time.RFC3339, getString(item, "created_at")); err == nil {
			video.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, getString(item, "updated_at")); err == nil {
			video.UpdatedAt = t
		}
		videos = append(videos, video)
	}
	return videos, nil
}

func getString(item map[string]types.AttributeValue, key string) string {
	if v, ok := item[key].(*types.AttributeValueMemberS); ok {
		return v.Value
	}
	return ""
}

func getInt(item map[string]types.AttributeValue, key string) int {
	if v, ok := item[key].(*types.AttributeValueMemberN); ok {
		val, _ := strconv.Atoi(v.Value)
		return val
	}
	return 0
}

func (r *videoRepositoryDynamoDB) Delete(ctx context.Context, videoID string, userID string) error {
	// Delete all chunks with pagination and BatchWriteItem
	var lastEvaluatedKey map[string]types.AttributeValue

	for {
		chunkQuery := &dynamodb.QueryInput{
			TableName:              aws.String(r.chunksTableName),
			KeyConditionExpression: aws.String("video_id = :vid"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":vid": &types.AttributeValueMemberS{Value: videoID},
			},
			ProjectionExpression: aws.String("video_id, part_number"),
		}
		if lastEvaluatedKey != nil {
			chunkQuery.ExclusiveStartKey = lastEvaluatedKey
		}

		chunkResp, err := r.db.Query(ctx, chunkQuery)
		if err != nil {
			return fmt.Errorf("failed to query chunks for deletion: %w", err)
		}

		// Build BatchWriteItem requests in groups of 25 (DynamoDB limit)
		for i := 0; i < len(chunkResp.Items); i += batchWriteMaxItems {
			end := i + batchWriteMaxItems
			if end > len(chunkResp.Items) {
				end = len(chunkResp.Items)
			}

			writeRequests := make([]types.WriteRequest, 0, end-i)
			for _, chunkItem := range chunkResp.Items[i:end] {
				writeRequests = append(writeRequests, types.WriteRequest{
					DeleteRequest: &types.DeleteRequest{
						Key: map[string]types.AttributeValue{
							"video_id":    chunkItem["video_id"],
							"part_number": chunkItem["part_number"],
						},
					},
				})
			}

			_, err := r.db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]types.WriteRequest{
					r.chunksTableName: writeRequests,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to batch delete chunks: %w", err)
			}
		}

		// Check if there are more pages
		if chunkResp.LastEvaluatedKey == nil {
			break
		}
		lastEvaluatedKey = chunkResp.LastEvaluatedKey
		log.Printf("Paginating chunk deletion for video %s, more pages remaining...", videoID)
	}

	// Delete the video
	_, err := r.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.videosTableName),
		Key: map[string]types.AttributeValue{
			"user_id":  &types.AttributeValueMemberS{Value: userID},
			"video_id": &types.AttributeValueMemberS{Value: videoID},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}

	return nil
}

func (r *videoRepositoryDynamoDB) Update(ctx context.Context, video *entity.Video) error {
	// Atualiza os campos do vídeo
	_, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.videosTableName),
		Key: map[string]types.AttributeValue{
			"user_id":  &types.AttributeValueMemberS{Value: video.User.ID},
			"video_id": &types.AttributeValueMemberS{Value: video.ID},
		},
		UpdateExpression: aws.String("SET #status = :status, download_url = :download_url, error_cause = :error_cause, updated_at = :updated_at"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":       &types.AttributeValueMemberS{Value: string(video.Status)},
			":download_url": &types.AttributeValueMemberS{Value: video.FileLocation.DownloadURL},
			":error_cause":  &types.AttributeValueMemberS{Value: video.ErrorCause},
			":updated_at":   &types.AttributeValueMemberS{Value: video.UpdatedAt.Format(time.RFC3339)},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update video: %w", err)
	}

	// Atualiza o status dos chunks
	for _, chunk := range video.Chunks {
		_, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String(r.chunksTableName),
			Key: map[string]types.AttributeValue{
				"video_id":    &types.AttributeValueMemberS{Value: video.ID},
				"part_number": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", chunk.PartNumber)},
			},
			UpdateExpression: aws.String("SET #status = :status"),
			ExpressionAttributeNames: map[string]string{
				"#status": "status",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":status": &types.AttributeValueMemberS{Value: string(chunk.Status)},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to update chunk %d: %w", chunk.PartNumber, err)
		}
	}
	return nil
}
