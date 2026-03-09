package use_case

import (
	"context"
	"fmt"

	"video_solicitation_microservice/internal/common/config/constant"
	"video_solicitation_microservice/internal/common/pkg/identity"
	"video_solicitation_microservice/internal/core/domain/port"
	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/dto"
	"video_solicitation_microservice/internal/core/exception"
)

type GetDownloadLink struct {
	videoRepo   port.VideoRepository
	blobStorage port.BlobStorage
}

func NewGetDownloadLink(videoRepo port.VideoRepository, blobStorage port.BlobStorage) *GetDownloadLink {
	return &GetDownloadLink{
		videoRepo:   videoRepo,
		blobStorage: blobStorage,
	}
}

func (uc *GetDownloadLink) Execute(ctx context.Context, videoID string) (*dto.DownloadLinkOutput, error) {
	if identity.IsNotValidUUID(videoID) {
		return nil, fmt.Errorf("%w: invalid video_id format", exception.ErrInvalidInput)
	}

	video, err := uc.videoRepo.FindByID(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if video == nil {
		return nil, exception.ErrVideoNotFound
	}

	if video.Status != value_object.VideoStatusCompleted {
		return nil, exception.ErrVideoNotCompleted
	}

	// Generate pre-signed download URL using the stored download path
	downloadKey := fmt.Sprintf("videos/%s/images_result.zip", video.ID)
	downloadURL, err := uc.blobStorage.GeneratePreSignedDownloadURL(
		ctx,
		video.FileLocation.BucketName,
		downloadKey,
		constant.PreSignedURLExpiration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate download URL: %w", err)
	}

	return &dto.DownloadLinkOutput{
		VideoID:     video.ID,
		Status:      string(video.Status),
		DownloadURL: downloadURL,
	}, nil
}
