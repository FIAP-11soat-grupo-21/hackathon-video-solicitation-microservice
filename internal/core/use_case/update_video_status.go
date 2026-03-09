package use_case

import (
	"context"
	"fmt"

	"video_solicitation_microservice/internal/core/domain/port"
	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/dto"
	"video_solicitation_microservice/internal/core/exception"
)

type UpdateVideoStatus struct {
	videoRepo port.VideoRepository
}

func NewUpdateVideoStatus(videoRepo port.VideoRepository) *UpdateVideoStatus {
	return &UpdateVideoStatus{
		videoRepo: videoRepo,
	}
}

func (uc *UpdateVideoStatus) Execute(ctx context.Context, input dto.UpdateVideoStatusInput) error {
	video, err := uc.videoRepo.FindByID(ctx, input.VideoID)
	if err != nil {
		return fmt.Errorf("failed to find video: %w", err)
	}
	if video == nil {
		return exception.ErrVideoNotFound
	}

	status := value_object.VideoStatus(input.Status)

	switch status {
	case value_object.VideoStatusCompleted:
		if err := video.Complete(input.DownloadURL); err != nil {
			return err
		}
	case value_object.VideoStatusError:
		if err := video.Fail(input.Cause); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: unsupported status transition to %s", exception.ErrInvalidStatusTransition, input.Status)
	}

	if err := uc.videoRepo.Update(ctx, video); err != nil {
		return fmt.Errorf("failed to update video: %w", err)
	}

	return nil
}
