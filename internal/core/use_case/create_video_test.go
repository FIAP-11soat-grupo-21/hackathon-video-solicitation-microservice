package use_case

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"video_solicitation_microservice/internal/common/config/constant"
	"video_solicitation_microservice/internal/core/domain/entity"
	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/dto"
	"video_solicitation_microservice/internal/core/exception"
)

func TestCreateVideoExecuteValidationErrors(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		mutate func(*dto.CreateVideoInput)
	}{
		{
			name: "rejects non positive numeric fields",
			mutate: func(input *dto.CreateVideoInput) {
				input.Metadata.DurationSeconds = 0
			},
		},
		{
			name: "rejects missing file name",
			mutate: func(input *dto.CreateVideoInput) {
				input.Metadata.FileName = ""
			},
		},
		{
			name: "rejects incomplete user",
			mutate: func(input *dto.CreateVideoInput) {
				input.User.Email = ""
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := validCreateVideoInput()
			tc.mutate(&input)

			output, err := NewCreateVideo(&videoRepoStub{}, &blobStorageStub{}).Execute(ctx, input)

			if output != nil {
				t.Fatalf("expected nil output, got %#v", output)
			}
			if !errors.Is(err, exception.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestCreateVideoExecuteReturnsSaveError(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("save failed")
	repo := &videoRepoStub{
		saveFunc: func(context.Context, *entity.Video) error {
			return repoErr
		},
	}

	output, err := NewCreateVideo(repo, &blobStorageStub{}).Execute(ctx, validCreateVideoInput())

	if output != nil {
		t.Fatalf("expected nil output, got %#v", output)
	}
	if err == nil || !strings.Contains(err.Error(), "failed to save video") || !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped save error, got %v", err)
	}
}

func TestCreateVideoExecuteReturnsUploadError(t *testing.T) {
	ctx := context.Background()
	blobErr := errors.New("upload failed")
	blob := &blobStorageStub{
		uploadFunc: func(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
			return "", blobErr
		},
	}

	output, err := NewCreateVideo(&videoRepoStub{}, blob).Execute(ctx, validCreateVideoInput())

	if output != nil {
		t.Fatalf("expected nil output, got %#v", output)
	}
	if err == nil || !strings.Contains(err.Error(), "failed to generate upload URL for chunk 1") || !errors.Is(err, blobErr) {
		t.Fatalf("expected wrapped upload error, got %v", err)
	}
	if len(blob.uploadCalls) != 1 {
		t.Fatalf("expected exactly one upload call before failure, got %d", len(blob.uploadCalls))
	}
}

func TestCreateVideoExecuteSuccess(t *testing.T) {
	ctx := context.Background()
	repo := &videoRepoStub{}
	blob := &blobStorageStub{
		uploadFunc: func(_ context.Context, _ string, key string, expiration time.Duration) (string, error) {
			if expiration != constant.PreSignedURLExpiration {
				return "", fmt.Errorf("unexpected expiration %s", expiration)
			}
			return "https://upload.local/" + key, nil
		},
	}

	output, err := NewCreateVideo(repo, blob).Execute(ctx, validCreateVideoInput())

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.savedVideo == nil {
		t.Fatal("expected video to be saved")
	}
	if output.VideoID != repo.savedVideo.ID {
		t.Fatalf("expected output video id %s, got %s", repo.savedVideo.ID, output.VideoID)
	}
	if output.Status != string(value_object.VideoStatusPending) {
		t.Fatalf("expected pending status, got %s", output.Status)
	}
	if len(output.Chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(output.Chunks))
	}
	if len(blob.uploadCalls) != len(output.Chunks) {
		t.Fatalf("expected one upload call per chunk, got %d", len(blob.uploadCalls))
	}
	if !strings.Contains(output.Chunks[0].UploadURL, repo.savedVideo.Chunks[0].VideoObjectID) {
		t.Fatalf("expected upload url to contain chunk key, got %s", output.Chunks[0].UploadURL)
	}
}