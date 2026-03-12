package use_case

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"video_solicitation_microservice/internal/core/domain/entity"
	"video_solicitation_microservice/internal/core/domain/value_object"
)

func TestGetVideosByUserExecutePassThrough(t *testing.T) {
	ctx := context.Background()
	expected := []*entity.Video{validVideo(value_object.VideoStatusPending)}
	repo := &videoRepoStub{
		findByUserIDFunc: func(_ context.Context, userID string) ([]*entity.Video, error) {
			if userID != "user-123" {
				return nil, fmt.Errorf("unexpected user %s", userID)
			}
			return expected, nil
		},
	}

	videos, err := NewGetVideosByUser(repo).Execute(ctx, "user-123")

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(videos) != 1 || videos[0] != expected[0] {
		t.Fatalf("expected passthrough result, got %#v", videos)
	}
}

func TestGetVideosByUserExecuteReturnsRepositoryError(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("query failed")
	repo := &videoRepoStub{
		findByUserIDFunc: func(context.Context, string) ([]*entity.Video, error) {
			return nil, repoErr
		},
	}

	videos, err := NewGetVideosByUser(repo).Execute(ctx, "user-123")

	if videos != nil {
		t.Fatalf("expected nil videos, got %#v", videos)
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}