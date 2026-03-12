package blob_storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"video_solicitation_microservice/internal/core/domain/port"
)

type s3Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
}

func NewS3Storage(client *s3.Client) port.BlobStorage {
	return &s3Storage{
		client:        client,
		presignClient: s3.NewPresignClient(client),
	}
}

func (s *s3Storage) GeneratePreSignedUploadURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error) {
	req, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", fmt.Errorf("failed to generate pre-signed upload URL: %w", err)
	}
	return req.URL, nil
}

func (s *s3Storage) GeneratePreSignedDownloadURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error) {
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", fmt.Errorf("failed to generate pre-signed download URL: %w", err)
	}
	return req.URL, nil
}

func (s *s3Storage) DeleteObjectsByPrefix(ctx context.Context, bucket, prefix string) error {
	listOutput, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return fmt.Errorf("failed to list objects with prefix %s: %w", prefix, err)
	}

	if len(listOutput.Contents) == 0 {
		return nil
	}

	objects := make([]s3types.ObjectIdentifier, 0, len(listOutput.Contents))
	for _, obj := range listOutput.Contents {
		objects = append(objects, s3types.ObjectIdentifier{
			Key: obj.Key,
		})
	}

	_, err = s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &s3types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete objects with prefix %s: %w", prefix, err)
	}

	return nil
}
