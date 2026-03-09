package port

import (
	"context"

	"video_solicitation_microservice/internal/core/dto"
)

type MessagePublisher interface {
	PublishAllChunksProcessed(ctx context.Context, payload dto.AllChunksProcessedEvent) error
}
