package use_case

import (
	"context"
	"fmt"
	"log"

	"video_solicitation_microservice/internal/common/config/constant"
	"video_solicitation_microservice/internal/core/domain/port"
	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/dto"
	"video_solicitation_microservice/internal/core/exception"
)

type UpdateChunkStatus struct {
	videoRepo port.VideoRepository
	publisher port.MessagePublisher
}

func NewUpdateChunkStatus(videoRepo port.VideoRepository, publisher port.MessagePublisher) *UpdateChunkStatus {
	return &UpdateChunkStatus{
		videoRepo: videoRepo,
		publisher: publisher,
	}
}

func (uc *UpdateChunkStatus) Execute(ctx context.Context, input dto.UpdateChunkStatusInput) error {
	video, err := uc.videoRepo.FindByID(ctx, input.VideoID)
	if err != nil {
		uc.publishErrorEvent(ctx, input, fmt.Sprintf("failed to find video: %v", err))
		return fmt.Errorf("failed to find video: %w", err)
	}
	if video == nil {
		return exception.ErrVideoNotFound
	}

	// Update chunk status
	chunkStatus := value_object.ChunkStatus(input.Status)
	if err := video.UpdateChunkStatus(input.ChunkPart, chunkStatus); err != nil {
		uc.publishErrorEvent(ctx, input, err.Error())
		return err
	}

	// If all chunks are processed, transition video to PROCESSING and publish event
	if video.AllChunksProcessed() {
		if err := video.TransitionTo(value_object.VideoStatusProcessing); err != nil {
			uc.publishErrorEvent(ctx, input, fmt.Sprintf("failed to transition video status: %v", err))
			return fmt.Errorf("failed to transition video status: %w", err)
		}

		event := dto.AllChunksProcessedEvent{
			VideoID: video.ID,
			User: dto.UserDTO{
				ID:    video.User.ID,
				Name:  video.User.Name,
				Email: video.User.Email,
			},
			BucketName:  video.FileLocation.BucketName,
			ImageFolder: video.FileLocation.ImageFolder,
		}

		if err := uc.publisher.PublishAllChunksProcessed(ctx, event); err != nil {
			log.Printf("WARN: failed to publish all-chunks-processed event for video %s: %v", video.ID, err)
			uc.publishErrorEvent(ctx, input, fmt.Sprintf("failed to publish event: %v", err))
			return fmt.Errorf("failed to publish event: %w", err)
		}
	}

	// Persist updated video
	if err := uc.videoRepo.Update(ctx, video); err != nil {
		uc.publishErrorEvent(ctx, input, fmt.Sprintf("failed to update video: %v", err))
		return fmt.Errorf("failed to update video: %w", err)
	}

	return nil
}

func (uc *UpdateChunkStatus) publishErrorEvent(ctx context.Context, input dto.UpdateChunkStatusInput, cause string) {
	event := dto.VideoProcessingErrorEvent{
		VideoID: input.VideoID,
		User: dto.UserDTO{
			ID:    input.User.ID,
			Name:  input.User.Name,
			Email: input.User.Email,
		},
		Status:        string(value_object.VideoStatusError),
		Cause:         cause,
		SystemTrigger: constant.SystemTriggerName,
	}

	if err := uc.publisher.PublishVideoProcessingError(ctx, event); err != nil {
		log.Printf("WARN: failed to publish video processing error event for video %s: %v", input.VideoID, err)
	}
}
