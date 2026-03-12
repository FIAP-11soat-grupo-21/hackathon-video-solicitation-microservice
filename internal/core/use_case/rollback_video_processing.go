package use_case

import (
	"context"
	"fmt"
	"log"

	"video_solicitation_microservice/internal/common/config/constant"
	"video_solicitation_microservice/internal/core/domain/port"
	"video_solicitation_microservice/internal/core/dto"
)

type RollbackVideoProcessing struct {
	videoRepo   port.VideoRepository
	blobStorage port.BlobStorage
}

func NewRollbackVideoProcessing(videoRepo port.VideoRepository, blobStorage port.BlobStorage) *RollbackVideoProcessing {
	return &RollbackVideoProcessing{
		videoRepo:   videoRepo,
		blobStorage: blobStorage,
	}
}

func (uc *RollbackVideoProcessing) Execute(ctx context.Context, input dto.VideoProcessingErrorEvent) error {
	// Ignore messages published by this service to avoid rollback loop
	if input.SystemTrigger == constant.SystemTriggerName {
		log.Printf("Ignoring rollback event from own service for video %s", input.VideoID)
		return nil
	}

	log.Printf("Starting rollback for video %s triggered by %s: %s", input.VideoID, input.SystemTrigger, input.Cause)

	// Find the video to get its metadata for cleanup
	video, err := uc.videoRepo.FindByID(ctx, input.VideoID)
	if err != nil {
		return fmt.Errorf("failed to find video for rollback: %w", err)
	}
	if video == nil {
		log.Printf("Video %s not found, rollback already completed or never existed", input.VideoID)
		return nil
	}

	// Delete S3 objects (video chunks and images)
	videoPrefix := fmt.Sprintf("videos/%s/", video.ID)
	if err := uc.blobStorage.DeleteObjectsByPrefix(ctx, video.FileLocation.BucketName, videoPrefix); err != nil {
		log.Printf("WARN: failed to delete S3 objects for video %s: %v", video.ID, err)
		// Continue with DB cleanup even if S3 deletion fails
	}

	// Delete video and chunks from database
	if err := uc.videoRepo.Delete(ctx, video.ID, video.User.ID); err != nil {
		return fmt.Errorf("failed to delete video %s from database: %w", video.ID, err)
	}

	log.Printf("Rollback completed for video %s", input.VideoID)
	return nil
}
