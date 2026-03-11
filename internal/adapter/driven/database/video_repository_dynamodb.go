package database

import (
	context "context"
	"fmt"
	"time"
	"video_solicitation_microservice/internal/core/domain/entity"
	"video_solicitation_microservice/internal/core/domain/port"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Repositório DynamoDB para vídeos

type videoRepositoryDynamoDB struct {
	db *dynamodb.Client
}

func NewVideoRepositoryDynamoDB(db *dynamodb.Client) port.VideoRepository {
	return &videoRepositoryDynamoDB{db: db}
}

func (r *videoRepositoryDynamoDB) Save(ctx context.Context, video *entity.Video) error {
	item := map[string]types.AttributeValue{
		"id":                 &types.AttributeValueMemberS{Value: video.ID},
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
		TableName: aws.String("Videos-05"),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to put video item: %w", err)
	}

	// TODO: Persistir chunks em tabela separada ou como atributo de lista, conforme modelagem

	return nil
}

func (r *videoRepositoryDynamoDB) FindByID(ctx context.Context, id string) (*entity.Video, error) {
	// TODO: Implementar GetItem ou Query para buscar vídeo por id
	return nil, nil
}

func (r *videoRepositoryDynamoDB) Update(ctx context.Context, video *entity.Video) error {
	// TODO: Implementar UpdateItem para atualizar status, download_url, error_cause, etc.
	return nil
}

// Adicione métodos auxiliares para conversão entre structs e atributos DynamoDB conforme necessário.
