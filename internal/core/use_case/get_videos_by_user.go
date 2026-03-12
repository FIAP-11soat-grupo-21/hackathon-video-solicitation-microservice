package use_case

import (
	"context"
	"video_solicitation_microservice/internal/core/domain/entity"
	"video_solicitation_microservice/internal/core/domain/port"
)

type GetVideosByUser struct {
	VideoRepository port.VideoRepository
}

func NewGetVideosByUser(repo port.VideoRepository) *GetVideosByUser {
	return &GetVideosByUser{VideoRepository: repo}
}

func (uc *GetVideosByUser) Execute(ctx context.Context, userID string) ([]*entity.Video, error) {
	return uc.VideoRepository.FindByUserID(ctx, userID)
}
