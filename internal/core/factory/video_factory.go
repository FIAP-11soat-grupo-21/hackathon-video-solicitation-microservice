package factory

import (
	"fmt"
	"time"

	"video_solicitation_microservice/internal/common/config/env"
	"video_solicitation_microservice/internal/common/pkg/identity"
	"video_solicitation_microservice/internal/core/domain/entity"
	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/dto"
)

// NewVideo creates the Aggregate Root with Metadata, FileLocation and calculated Chunks.
func NewVideo(input dto.CreateVideoInput) *entity.Video {
	cfg := env.LoadConfig()
	id := identity.NewUUIDV7()
	now := time.Now()

	video := &entity.Video{
		ID: id,
		User: entity.User{
			ID:    input.User.ID,
			Name:  input.User.Name,
			Email: input.User.Email,
		},
		Metadata: entity.Metadata{
			FileName:        input.Metadata.FileName,
			DurationSeconds: input.Metadata.DurationSeconds,
			SizeBytes:       input.Metadata.SizeBytes,
		},
		Status: value_object.VideoStatusPending,
		FileLocation: entity.FileLocation{
			BucketName:       cfg.AWS.S3.BucketName,
			VideoChunkFolder: fmt.Sprintf("videos/%s/chunks/", id),
			ImageFolder:      fmt.Sprintf("videos/%s/images/", id),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	video.CalculateChunks(input.FramesPerSecond)
	return video
}
