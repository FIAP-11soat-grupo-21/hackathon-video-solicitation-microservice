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

func TestUpdateVideoStatusExecuteScenarios(t *testing.T) {
	ctx := context.Background()
	baseInput := dto.UpdateVideoStatusInput{
		VideoID: identity.NewUUIDV7(),
		User: dto.UserDTO{
			ID:    identity.NewUUIDV7(),
			Name:  "Mateus",
			Email: "mateus@example.com",
		},
	}
	findErr := errors.New("find failed")
	updateErr := errors.New("update failed")

	tests := []struct {
		name       string
		input      dto.UpdateVideoStatusInput
		repo       *videoRepoStub
		publisher  *publisherStub
		assertErr  func(*testing.T, error)
		assertSide func(*testing.T, *videoRepoStub, *publisherStub)
	}{
		{
			name:  "publishes error when repository lookup fails",
			input: withStatus(baseInput, string(value_object.VideoStatusProcessing)),
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
			input:     withStatus(baseInput, string(value_object.VideoStatusProcessing)),
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
					t.Fatal("expected no repository update and no error event")
				}
			},
		},
		{
			name:  "transitions to processing",
			input: withStatus(baseInput, string(value_object.VideoStatusProcessing)),
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				return validVideo(value_object.VideoStatusPending), nil
			}},
			publisher: &publisherStub{},
			assertSide: func(t *testing.T, repo *videoRepoStub, publisher *publisherStub) {
				t.Helper()
				if repo.updatedVideo == nil || repo.updatedVideo.Status != value_object.VideoStatusProcessing {
					t.Fatalf("expected processing status to be persisted, got %#v", repo.updatedVideo)
				}
				if len(publisher.errorEvents) != 0 {
					t.Fatalf("expected no error event, got %d", len(publisher.errorEvents))
				}
			},
		},
		{
			name: "completes video and stores download url",
			input: func() dto.UpdateVideoStatusInput {
				input := withStatus(baseInput, string(value_object.VideoStatusCompleted))
				input.DownloadURL = "https://download.local/file.zip"
				return input
			}(),
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				return validVideo(value_object.VideoStatusProcessing), nil
			}},
			publisher: &publisherStub{},
			assertSide: func(t *testing.T, repo *videoRepoStub, _ *publisherStub) {
				t.Helper()
				if repo.updatedVideo == nil || repo.updatedVideo.FileLocation.DownloadURL != "https://download.local/file.zip" {
					t.Fatalf("expected download url to be set, got %#v", repo.updatedVideo)
				}
			},
		},
		{
			name: "fails video and stores cause",
			input: func() dto.UpdateVideoStatusInput {
				input := withStatus(baseInput, string(value_object.VideoStatusError))
				input.Cause = "transcoding failed"
				return input
			}(),
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				return validVideo(value_object.VideoStatusPending), nil
			}},
			publisher: &publisherStub{},
			assertSide: func(t *testing.T, repo *videoRepoStub, _ *publisherStub) {
				t.Helper()
				if repo.updatedVideo == nil || repo.updatedVideo.ErrorCause != "transcoding failed" {
					t.Fatalf("expected error cause to be set, got %#v", repo.updatedVideo)
				}
			},
		},
		{
			name:  "rejects unsupported status",
			input: withStatus(baseInput, "UNKNOWN"),
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				return validVideo(value_object.VideoStatusPending), nil
			}},
			publisher: &publisherStub{},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if err == nil || !errors.Is(err, exception.ErrInvalidStatusTransition) {
					t.Fatalf("expected invalid status transition error, got %v", err)
				}
			},
			assertSide: func(t *testing.T, repo *videoRepoStub, publisher *publisherStub) {
				t.Helper()
				if repo.updatedVideo != nil {
					t.Fatal("expected no repository update")
				}
				if len(publisher.errorEvents) != 1 {
					t.Fatalf("expected one error event, got %d", len(publisher.errorEvents))
				}
			},
		},
		{
			name:  "publishes error when repository update fails",
			input: withStatus(baseInput, string(value_object.VideoStatusProcessing)),
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
			err := NewUpdateVideoStatus(tc.repo, tc.publisher).Execute(ctx, tc.input)

			if tc.assertErr != nil {
				tc.assertErr(t, err)
			} else if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			tc.assertSide(t, tc.repo, tc.publisher)
		})
	}
}
