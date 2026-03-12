package use_case

import (
	"context"
	"errors"
	"strings"
	"testing"

	"video_solicitation_microservice/internal/common/pkg/identity"
	"video_solicitation_microservice/internal/core/domain/entity"
	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/dto"
	"video_solicitation_microservice/internal/core/exception"
)

func TestUpdateChunkStatusExecuteScenarios(t *testing.T) {
	ctx := context.Background()
	baseInput := dto.UpdateChunkStatusInput{
		VideoID:   identity.NewUUIDV7(),
		ChunkPart: 2,
		Status:    string(value_object.ChunkStatusProcessed),
		User: dto.UserDTO{
			ID:    identity.NewUUIDV7(),
			Name:  "Mateus",
			Email: "mateus@example.com",
		},
	}
	findErr := errors.New("find failed")
	publishErr := errors.New("publish failed")
	updateErr := errors.New("update failed")

	tests := []struct {
		name       string
		repo       *videoRepoStub
		publisher  *publisherStub
		assertErr  func(*testing.T, error)
		assertSide func(*testing.T, *videoRepoStub, *publisherStub)
	}{
		{
			name: "publishes error when repository lookup fails",
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				return nil, findErr
			}},
			publisher: &publisherStub{},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if err == nil || !strings.Contains(err.Error(), "failed to find video") || !errors.Is(err, findErr) {
					t.Fatalf("expected wrapped find error, got %v", err)
				}
			},
			assertSide: func(t *testing.T, _ *videoRepoStub, publisher *publisherStub) {
				t.Helper()
				if len(publisher.errorEvents) != 1 {
					t.Fatalf("expected one error event, got %d", len(publisher.errorEvents))
				}
			},
		},
		{
			name:      "returns not found when video is missing",
			repo:      &videoRepoStub{},
			publisher: &publisherStub{},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, exception.ErrVideoNotFound) {
					t.Fatalf("expected ErrVideoNotFound, got %v", err)
				}
			},
			assertSide: func(t *testing.T, repo *videoRepoStub, publisher *publisherStub) {
				t.Helper()
				if repo.updatedVideo != nil || len(publisher.errorEvents) != 0 {
					t.Fatal("expected no update and no error event")
				}
			},
		},
		{
			name: "publishes error when chunk is missing",
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				video := validVideo(value_object.VideoStatusPending)
				video.Chunks = video.Chunks[:1]
				return video, nil
			}},
			publisher: &publisherStub{},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, exception.ErrChunkNotFound) {
					t.Fatalf("expected ErrChunkNotFound, got %v", err)
				}
			},
			assertSide: func(t *testing.T, _ *videoRepoStub, publisher *publisherStub) {
				t.Helper()
				if len(publisher.errorEvents) != 1 {
					t.Fatalf("expected one error event, got %d", len(publisher.errorEvents))
				}
			},
		},
		{
			name: "transitions and publishes when all chunks are processed",
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				video := validVideo(value_object.VideoStatusPending)
				video.Chunks[0].Status = value_object.ChunkStatusProcessed
				return video, nil
			}},
			publisher: &publisherStub{},
			assertSide: func(t *testing.T, repo *videoRepoStub, publisher *publisherStub) {
				t.Helper()
				if repo.updatedVideo == nil {
					t.Fatal("expected updated video to be persisted")
				}
				if repo.updatedVideo.Status != value_object.VideoStatusProcessing {
					t.Fatalf("expected processing status, got %s", repo.updatedVideo.Status)
				}
				if len(publisher.allChunksProcessedEvents) != 1 {
					t.Fatalf("expected one all chunks processed event, got %d", len(publisher.allChunksProcessedEvents))
				}
				if len(publisher.errorEvents) != 0 {
					t.Fatalf("expected no error events, got %d", len(publisher.errorEvents))
				}
			},
		},
		{
			name: "returns publish error and emits processing error",
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				video := validVideo(value_object.VideoStatusPending)
				video.Chunks[0].Status = value_object.ChunkStatusProcessed
				return video, nil
			}},
			publisher: &publisherStub{allChunksProcessedFunc: func(context.Context, dto.AllChunksProcessedEvent) error {
				return publishErr
			}},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if err == nil || !strings.Contains(err.Error(), "failed to publish event") || !errors.Is(err, publishErr) {
					t.Fatalf("expected wrapped publish error, got %v", err)
				}
			},
			assertSide: func(t *testing.T, repo *videoRepoStub, publisher *publisherStub) {
				t.Helper()
				if repo.updatedVideo != nil {
					t.Fatal("expected no repository update after publish failure")
				}
				if len(publisher.errorEvents) != 1 {
					t.Fatalf("expected one error event, got %d", len(publisher.errorEvents))
				}
			},
		},
		{
			name: "returns update error and emits processing error",
			repo: &videoRepoStub{
				findByIDFunc: func(context.Context, string) (*entity.Video, error) {
					return validVideo(value_object.VideoStatusPending), nil
				},
				updateFunc: func(context.Context, *entity.Video) error {
					return updateErr
				},
			},
			publisher: &publisherStub{},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if err == nil || !strings.Contains(err.Error(), "failed to update video") || !errors.Is(err, updateErr) {
					t.Fatalf("expected wrapped update error, got %v", err)
				}
			},
			assertSide: func(t *testing.T, _ *videoRepoStub, publisher *publisherStub) {
				t.Helper()
				if len(publisher.errorEvents) != 1 {
					t.Fatalf("expected one error event, got %d", len(publisher.errorEvents))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := NewUpdateChunkStatus(tc.repo, tc.publisher).Execute(ctx, baseInput)

			if tc.assertErr != nil {
				tc.assertErr(t, err)
			} else if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			tc.assertSide(t, tc.repo, tc.publisher)
		})
	}
}
