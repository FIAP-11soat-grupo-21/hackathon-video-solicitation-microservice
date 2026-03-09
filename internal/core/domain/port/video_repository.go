package port

import (
	"context"

	"video_solicitation_microservice/internal/core/domain/entity"
)

type VideoRepository interface {
	Save(ctx context.Context, video *entity.Video) error
	FindByID(ctx context.Context, id string) (*entity.Video, error)
	Update(ctx context.Context, video *entity.Video) error
}
