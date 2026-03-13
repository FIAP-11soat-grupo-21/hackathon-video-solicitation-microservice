package use_case

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"video_solicitation_microservice/internal/common/config/constant"
	"video_solicitation_microservice/internal/common/pkg/identity"
	"video_solicitation_microservice/internal/core/domain/entity"
	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/dto"
	"video_solicitation_microservice/internal/core/exception"
)

func TestGetDownloadLinkExecuteScenarios(t *testing.T) {
	ctx := context.Background()
	validID := identity.NewUUIDV7()
	repoErr := errors.New("find failed")
	downloadErr := errors.New("download failed")

	tests := []struct {
		name      string
		videoID   string
		repo      *videoRepoStub
		blob      *blobStorageStub
		assertErr func(*testing.T, error)
		assertOK  func(*testing.T, *dto.DownloadLinkOutput, *blobStorageStub)
	}{
		{
			name:    "rejects invalid uuid",
			videoID: "invalid-id",
			repo:    &videoRepoStub{},
			blob:    &blobStorageStub{},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, exception.ErrInvalidInput) {
					t.Fatalf("expected ErrInvalidInput, got %v", err)
				}
			},
		},
		{
			name:    "propagates repository error",
			videoID: validID,
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				return nil, repoErr
			}},
			blob: &blobStorageStub{},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, repoErr) {
					t.Fatalf("expected repository error, got %v", err)
				}
			},
		},
		{
			name:    "returns video not found",
			videoID: validID,
			repo:    &videoRepoStub{},
			blob:    &blobStorageStub{},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, exception.ErrVideoNotFound) {
					t.Fatalf("expected ErrVideoNotFound, got %v", err)
				}
			},
		},
		{
			name:    "rejects incomplete video",
			videoID: validID,
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				return validVideo(value_object.VideoStatusProcessing), nil
			}},
			blob: &blobStorageStub{},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, exception.ErrVideoNotCompleted) {
					t.Fatalf("expected ErrVideoNotCompleted, got %v", err)
				}
			},
		},
		{
			name:    "returns download generation error",
			videoID: validID,
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				return validVideo(value_object.VideoStatusCompleted), nil
			}},
			blob: &blobStorageStub{downloadFunc: func(context.Context, string, string, time.Duration) (string, error) {
				return "", downloadErr
			}},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if err == nil || !strings.Contains(err.Error(), "failed to generate download URL") || !errors.Is(err, downloadErr) {
					t.Fatalf("expected wrapped download error, got %v", err)
				}
			},
		},
		{
			name:    "returns pre signed download url",
			videoID: validID,
			repo: &videoRepoStub{findByIDFunc: func(context.Context, string) (*entity.Video, error) {
				video := validVideo(value_object.VideoStatusCompleted)
				video.ID = validID
				video.FileLocation.BucketName = "download-bucket"
				return video, nil
			}},
			blob: &blobStorageStub{downloadFunc: func(_ context.Context, bucket, key string, expiration time.Duration) (string, error) {
				if bucket != "download-bucket" {
					return "", fmt.Errorf("unexpected bucket %s", bucket)
				}
				if expiration != constant.PreSignedURLExpiration {
					return "", fmt.Errorf("unexpected expiration %s", expiration)
				}
				return "https://download.local/" + key, nil
			}},
			assertOK: func(t *testing.T, output *dto.DownloadLinkOutput, blob *blobStorageStub) {
				t.Helper()
				if output.VideoID != validID {
					t.Fatalf("expected video id %s, got %s", validID, output.VideoID)
				}
				if output.Status != string(value_object.VideoStatusCompleted) {
					t.Fatalf("expected completed status, got %s", output.Status)
				}
				if output.DownloadURL != "https://download.local/videos/"+validID+"/images_result.zip" {
					t.Fatalf("unexpected download url %s", output.DownloadURL)
				}
				if len(blob.downloadCalls) != 1 {
					t.Fatalf("expected one download call, got %d", len(blob.downloadCalls))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output, err := NewGetDownloadLink(tc.repo, tc.blob).Execute(ctx, tc.videoID)

			if tc.assertErr != nil {
				if output != nil {
					t.Fatalf("expected nil output, got %#v", output)
				}
				tc.assertErr(t, err)
				return
			}

			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			tc.assertOK(t, output, tc.blob)
		})
	}
}