package port

import (
	"context"
	"time"
)

type BlobStorage interface {
	GeneratePreSignedUploadURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error)
	GeneratePreSignedDownloadURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error)
}
