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

type UpdateVideoStatus struct {
	videoRepo port.VideoRepository
	publisher port.MessagePublisher
}

func NewUpdateVideoStatus(videoRepo port.VideoRepository, publisher port.MessagePublisher) *UpdateVideoStatus {
	return &UpdateVideoStatus{
		videoRepo: videoRepo,
		publisher: publisher,
	}
}

func (uc *UpdateVideoStatus) Execute(ctx context.Context, input dto.UpdateVideoStatusInput) error {
	video, err := uc.videoRepo.FindByID(ctx, input.VideoID)
	if err != nil {
		uc.publishErrorEvent(ctx, input, fmt.Sprintf("failed to find video: %v", err))
		return fmt.Errorf("failed to find video: %w", err)
	}
	if video == nil {
		return exception.ErrVideoNotFound
	}

	status := value_object.VideoStatus(input.Status)

	switch status {
	case value_object.VideoStatusProcessing:
		if err := video.TransitionTo(value_object.VideoStatusProcessing); err != nil {
			uc.publishErrorEvent(ctx, input, err.Error())
			return err
		}
	case value_object.VideoStatusCompleted:
		if err := video.Complete(input.DownloadURL); err != nil {
			uc.publishErrorEvent(ctx, input, err.Error())
			return err
		}
	case value_object.VideoStatusError:
		if err := video.Fail(input.Cause); err != nil {
			uc.publishErrorEvent(ctx, input, err.Error())
			return err
		}
	default:
		uc.publishErrorEvent(ctx, input, fmt.Sprintf("unsupported status transition to %s", input.Status))
		return fmt.Errorf("%w: unsupported status transition to %s", exception.ErrInvalidStatusTransition, input.Status)
	}

	if err := uc.videoRepo.Update(ctx, video); err != nil {
		uc.publishErrorEvent(ctx, input, fmt.Sprintf("failed to update video: %v", err))
		return fmt.Errorf("failed to update video: %w", err)
	}

	return nil
}

func (uc *UpdateVideoStatus) publishErrorEvent(ctx context.Context, input dto.UpdateVideoStatusInput, cause string) {
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
