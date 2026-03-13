package entity

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/exception"
)

func newTestVideo() *Video {
	return &Video{
		ID: "video-123",
		Metadata: Metadata{
			DurationSeconds: 400,
		},
		Status: value_object.VideoStatusPending,
		Chunks: []Chunk{
			{PartNumber: 1, Status: value_object.ChunkStatusPending},
			{PartNumber: 2, Status: value_object.ChunkStatusPending},
		},
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}
}

func TestVideoCalculateChunks(t *testing.T) {
	video := &Video{
		ID: "video-123",
		Metadata: Metadata{
			DurationSeconds: 400,
		},
	}

	video.CalculateChunks(30)

	if len(video.Chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(video.Chunks))
	}
	expected := []struct {
		partNumber int
		startTime  int
		endTime    int
		objectID   string
	}{
		{1, 0, 180, "videos/video-123/chunks/part_1.mp4"},
		{2, 180, 360, "videos/video-123/chunks/part_2.mp4"},
		{3, 360, 400, "videos/video-123/chunks/part_3.mp4"},
	}
	for index, chunk := range video.Chunks {
		if chunk.PartNumber != expected[index].partNumber || chunk.StartTime != expected[index].startTime || chunk.EndTime != expected[index].endTime {
			t.Fatalf("unexpected chunk at index %d: %#v", index, chunk)
		}
		if chunk.FramePerSecond != 30 {
			t.Fatalf("expected fps 30, got %d", chunk.FramePerSecond)
		}
		if chunk.Status != value_object.ChunkStatusPending {
			t.Fatalf("expected pending status, got %s", chunk.Status)
		}
		if chunk.VideoObjectID != expected[index].objectID {
			t.Fatalf("expected object id %s, got %s", expected[index].objectID, chunk.VideoObjectID)
		}
	}
}

func TestVideoUpdateChunkStatus(t *testing.T) {
	video := newTestVideo()
	before := video.UpdatedAt

	err := video.UpdateChunkStatus(2, value_object.ChunkStatusProcessed)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if video.Chunks[1].Status != value_object.ChunkStatusProcessed {
		t.Fatalf("expected processed status, got %s", video.Chunks[1].Status)
	}
	if !video.UpdatedAt.After(before) {
		t.Fatalf("expected updated timestamp to move forward, before=%v after=%v", before, video.UpdatedAt)
	}
}

func TestVideoUpdateChunkStatusReturnsNotFound(t *testing.T) {
	video := newTestVideo()

	err := video.UpdateChunkStatus(99, value_object.ChunkStatusProcessed)

	if !errors.Is(err, exception.ErrChunkNotFound) {
		t.Fatalf("expected ErrChunkNotFound, got %v", err)
	}
}

func TestVideoAllChunksProcessed(t *testing.T) {
	tests := []struct {
		name   string
		chunks []Chunk
		want   bool
	}{
		{name: "empty list is false", chunks: nil, want: false},
		{name: "pending chunk is false", chunks: []Chunk{{Status: value_object.ChunkStatusProcessed}, {Status: value_object.ChunkStatusPending}}, want: false},
		{name: "all processed is true", chunks: []Chunk{{Status: value_object.ChunkStatusProcessed}, {Status: value_object.ChunkStatusProcessed}}, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			video := &Video{Chunks: tc.chunks}
			if got := video.AllChunksProcessed(); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}

func TestVideoTransitionTo(t *testing.T) {
	tests := []struct {
		name        string
		from        value_object.VideoStatus
		to          value_object.VideoStatus
		shouldError bool
	}{
		{name: "pending to processing", from: value_object.VideoStatusPending, to: value_object.VideoStatusProcessing},
		{name: "processing to completed", from: value_object.VideoStatusProcessing, to: value_object.VideoStatusCompleted},
		{name: "pending to error", from: value_object.VideoStatusPending, to: value_object.VideoStatusError},
		{name: "processing to error", from: value_object.VideoStatusProcessing, to: value_object.VideoStatusError},
		{name: "pending to completed is invalid", from: value_object.VideoStatusPending, to: value_object.VideoStatusCompleted, shouldError: true},
		{name: "completed to processing is invalid", from: value_object.VideoStatusCompleted, to: value_object.VideoStatusProcessing, shouldError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			video := newTestVideo()
			video.Status = tc.from
			before := video.UpdatedAt

			err := video.TransitionTo(tc.to)

			if tc.shouldError {
				if !errors.Is(err, exception.ErrInvalidStatusTransition) {
					t.Fatalf("expected invalid transition error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if video.Status != tc.to {
				t.Fatalf("expected status %s, got %s", tc.to, video.Status)
			}
			if !video.UpdatedAt.After(before) {
				t.Fatalf("expected updated timestamp to move forward, before=%v after=%v", before, video.UpdatedAt)
			}
		})
	}
}

func TestVideoComplete(t *testing.T) {
	video := newTestVideo()
	video.Status = value_object.VideoStatusProcessing

	err := video.Complete("https://download.local/archive.zip")

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if video.Status != value_object.VideoStatusCompleted {
		t.Fatalf("expected completed status, got %s", video.Status)
	}
	if video.FileLocation.DownloadURL != "https://download.local/archive.zip" {
		t.Fatalf("unexpected download url %s", video.FileLocation.DownloadURL)
	}
}

func TestVideoCompleteReturnsTransitionError(t *testing.T) {
	video := newTestVideo()

	err := video.Complete("https://download.local/archive.zip")

	if !errors.Is(err, exception.ErrInvalidStatusTransition) {
		t.Fatalf("expected invalid transition error, got %v", err)
	}
}

func TestVideoFail(t *testing.T) {
	video := newTestVideo()

	err := video.Fail("processor failed")

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if video.Status != value_object.VideoStatusError {
		t.Fatalf("expected error status, got %s", video.Status)
	}
	if video.ErrorCause != "processor failed" {
		t.Fatalf("unexpected cause %s", video.ErrorCause)
	}
}

func TestVideoFailReturnsTransitionError(t *testing.T) {
	video := newTestVideo()
	video.Status = value_object.VideoStatusCompleted

	err := video.Fail("processor failed")

	if !errors.Is(err, exception.ErrInvalidStatusTransition) {
		t.Fatalf("expected invalid transition error, got %v", err)
	}
	if video.ErrorCause != "" {
		t.Fatalf("expected empty cause on failure, got %s", video.ErrorCause)
	}
}

func TestVideoTransitionErrorMessageIncludesStates(t *testing.T) {
	video := newTestVideo()
	video.Status = value_object.VideoStatusCompleted

	err := video.TransitionTo(value_object.VideoStatusProcessing)

	if err == nil || !stringsContainAll(err.Error(), string(value_object.VideoStatusCompleted), string(value_object.VideoStatusProcessing)) {
		t.Fatalf("expected error to mention states, got %v", err)
	}
}

func stringsContainAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !contains(value, part) {
			return false
		}
	}
	return true
}

func contains(value, part string) bool {
	return fmt.Sprintf("%s", value) != "" && len(part) > 0 && (len(value) >= len(part) && (value == part || len(value) > len(part) && (func() bool { return stringIndex(value, part) >= 0 })()))
}

func stringIndex(value, part string) int {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return i
		}
	}
	return -1
}
