package factory

import (
	"strings"
	"testing"
	"time"

	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/dto"
)

func TestNewVideoBuildsAggregateRoot(t *testing.T) {
	before := time.Now()
	input := dto.CreateVideoInput{
		User: dto.UserDTO{
			ID:    "user-123",
			Name:  "Mateus",
			Email: "mateus@example.com",
		},
		Metadata: dto.MetadataDTO{
			FileName:        "video.mp4",
			DurationSeconds: 400,
			SizeBytes:       2048,
		},
		FramesPerSecond: 25,
	}

	video := NewVideo(input)
	after := time.Now()

	if video.ID == "" {
		t.Fatal("expected generated id")
	}
	if video.User.ID != input.User.ID || video.User.Name != input.User.Name || video.User.Email != input.User.Email {
		t.Fatalf("unexpected user copied into aggregate: %#v", video.User)
	}
	if video.Metadata.FileName != input.Metadata.FileName || video.Metadata.DurationSeconds != input.Metadata.DurationSeconds || video.Metadata.SizeBytes != input.Metadata.SizeBytes {
		t.Fatalf("unexpected metadata copied into aggregate: %#v", video.Metadata)
	}
	if video.Status != value_object.VideoStatusPending {
		t.Fatalf("expected pending status, got %s", video.Status)
	}
	if video.FileLocation.BucketName == "" {
		t.Fatal("expected bucket name to be populated")
	}
	if !strings.HasPrefix(video.FileLocation.VideoChunkFolder, "videos/"+video.ID+"/chunks/") {
		t.Fatalf("unexpected chunk folder %s", video.FileLocation.VideoChunkFolder)
	}
	if !strings.HasPrefix(video.FileLocation.ImageFolder, "videos/"+video.ID+"/images/") {
		t.Fatalf("unexpected image folder %s", video.FileLocation.ImageFolder)
	}
	if video.CreatedAt.Before(before) || video.CreatedAt.After(after) {
		t.Fatalf("expected created time between %v and %v, got %v", before, after, video.CreatedAt)
	}
	if !video.UpdatedAt.Equal(video.CreatedAt) {
		t.Fatalf("expected updated time equal created time, got created=%v updated=%v", video.CreatedAt, video.UpdatedAt)
	}
	if len(video.Chunks) != 3 {
		t.Fatalf("expected calculated chunks, got %d", len(video.Chunks))
	}
	if video.Chunks[0].FramePerSecond != 25 {
		t.Fatalf("expected fps propagated to chunks, got %d", video.Chunks[0].FramePerSecond)
	}
	if !strings.Contains(video.Chunks[0].VideoObjectID, video.ID) {
		t.Fatalf("expected chunk object id to contain video id, got %s", video.Chunks[0].VideoObjectID)
	}
}
