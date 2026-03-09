package use_case

import (
	"context"
	"fmt"

	"video_solicitation_microservice/internal/common/config/constant"
	"video_solicitation_microservice/internal/core/domain/port"
	"video_solicitation_microservice/internal/core/dto"
	"video_solicitation_microservice/internal/core/exception"
	"video_solicitation_microservice/internal/core/factory"
)

type CreateVideo struct {
	videoRepo   port.VideoRepository
	blobStorage port.BlobStorage
}

func NewCreateVideo(videoRepo port.VideoRepository, blobStorage port.BlobStorage) *CreateVideo {
	return &CreateVideo{
		videoRepo:   videoRepo,
		blobStorage: blobStorage,
	}
}

func (uc *CreateVideo) Execute(ctx context.Context, input dto.CreateVideoInput) (*dto.CreateVideoOutput, error) {
	// Validate input
	if input.Metadata.DurationSeconds <= 0 || input.Metadata.SizeBytes <= 0 || input.FramesPerSecond <= 0 {
		return nil, fmt.Errorf("%w: duration_seconds, size_bytes and frames_per_second must be positive", exception.ErrInvalidInput)
	}
	if input.Metadata.FileName == "" {
		return nil, fmt.Errorf("%w: file_name is required", exception.ErrInvalidInput)
	}
	if input.User.ID == "" || input.User.Name == "" || input.User.Email == "" {
		return nil, fmt.Errorf("%w: user id, name and email are required", exception.ErrInvalidInput)
	}

	// Create aggregate root with calculated chunks
	video := factory.NewVideo(input)

	// Persist video
	if err := uc.videoRepo.Save(ctx, video); err != nil {
		return nil, fmt.Errorf("failed to save video: %w", err)
	}

	// Generate pre-signed upload URLs for each chunk
	chunkOutputs := make([]dto.ChunkOutputDTO, 0, len(video.Chunks))
	for _, chunk := range video.Chunks {
		uploadURL, err := uc.blobStorage.GeneratePreSignedUploadURL(
			ctx,
			video.FileLocation.BucketName,
			chunk.VideoObjectID,
			constant.PreSignedURLExpiration,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to generate upload URL for chunk %d: %w", chunk.PartNumber, err)
		}

		chunkOutputs = append(chunkOutputs, dto.ChunkOutputDTO{
			PartNumber: chunk.PartNumber,
			StartTime:  chunk.StartTime,
			EndTime:    chunk.EndTime,
			UploadURL:  uploadURL,
		})
	}

	return &dto.CreateVideoOutput{
		VideoID: video.ID,
		Status:  string(video.Status),
		Chunks:  chunkOutputs,
	}, nil
}
