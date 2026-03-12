package use_case

import (
	"context"
	"errors"
	"strings"
	"testing"

	"video_solicitation_microservice/internal/common/config/constant"
	"video_solicitation_microservice/internal/common/pkg/identity"
	"video_solicitation_microservice/internal/core/domain/entity"
	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/dto"
)

func TestRollbackVideoProcessingIgnoresOwnEvents(t *testing.T) {
	ctx := context.Background()
	repo := &videoRepoStub{}
	blob := &blobStorageStub{}
	input := dto.VideoProcessingErrorEvent{
		VideoID:       identity.NewUUIDV7(),
		SystemTrigger: constant.SystemTriggerName,
	}

	err := NewRollbackVideoProcessing(repo, blob).Execute(ctx, input)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.findCalls != 0 {
		t.Fatalf("expected repo not to be queried, got %d calls", repo.findCalls)
	}
	if blob.deletePrefix != "" {
		t.Fatalf("expected no blob deletion, got prefix %s", blob.deletePrefix)
	}
}

func TestRollbackVideoProcessingReturnsFindError(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("find failed")
	repo := &videoRepoStub{
		findByIDFunc: func(context.Context, string) (*entity.Video, error) {
			return nil, repoErr
		},
	}

	err := NewRollbackVideoProcessing(repo, &blobStorageStub{}).Execute(ctx, dto.VideoProcessingErrorEvent{VideoID: identity.NewUUIDV7(), SystemTrigger: "external"})

	if err == nil || !strings.Contains(err.Error(), "failed to find video for rollback") || !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped find error, got %v", err)
	}
}

func TestRollbackVideoProcessingReturnsNilWhenVideoIsMissing(t *testing.T) {
	ctx := context.Background()
	repo := &videoRepoStub{}
	blob := &blobStorageStub{}

	err := NewRollbackVideoProcessing(repo, blob).Execute(ctx, dto.VideoProcessingErrorEvent{VideoID: identity.NewUUIDV7(), SystemTrigger: "external"})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if blob.deletePrefix != "" {
		t.Fatalf("expected no blob deletion, got prefix %s", blob.deletePrefix)
	}
}

func TestRollbackVideoProcessingContinuesWhenBlobDeletionFails(t *testing.T) {
	ctx := context.Background()
	video := validVideo(value_object.VideoStatusProcessing)
	blobErr := errors.New("delete failed")
	repo := &videoRepoStub{
		findByIDFunc: func(context.Context, string) (*entity.Video, error) {
			return video, nil
		},
	}
	blob := &blobStorageStub{
		deleteFunc: func(context.Context, string, string) error {
			return blobErr
		},
	}

	err := NewRollbackVideoProcessing(repo, blob).Execute(ctx, dto.VideoProcessingErrorEvent{VideoID: video.ID, SystemTrigger: "external"})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.deleteVideo != video.ID || repo.deleteUser != video.User.ID {
		t.Fatalf("expected delete to use video and user ids, got video=%s user=%s", repo.deleteVideo, repo.deleteUser)
	}
	if blob.deletePrefix != "videos/"+video.ID+"/" {
		t.Fatalf("unexpected delete prefix %s", blob.deletePrefix)
	}
}

func TestRollbackVideoProcessingReturnsDeleteError(t *testing.T) {
	ctx := context.Background()
	video := validVideo(value_object.VideoStatusProcessing)
	deleteErr := errors.New("db delete failed")
	repo := &videoRepoStub{
		findByIDFunc: func(context.Context, string) (*entity.Video, error) {
			return video, nil
		},
		deleteFunc: func(context.Context, string, string) error {
			return deleteErr
		},
	}

	err := NewRollbackVideoProcessing(repo, &blobStorageStub{}).Execute(ctx, dto.VideoProcessingErrorEvent{VideoID: video.ID, SystemTrigger: "external"})

	if err == nil || !strings.Contains(err.Error(), "failed to delete video") || !errors.Is(err, deleteErr) {
		t.Fatalf("expected wrapped delete error, got %v", err)
	}
}