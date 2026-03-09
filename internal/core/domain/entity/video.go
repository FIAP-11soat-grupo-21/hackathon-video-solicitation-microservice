package entity

import (
	"fmt"
	"math"
	"time"

	"video_solicitation_microservice/internal/common/config/constant"
	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/exception"
)

type Video struct {
	ID           string
	User         User
	Metadata     Metadata
	Status       value_object.VideoStatus
	Chunks       []Chunk
	FileLocation FileLocation
	ErrorCause   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CalculateChunks computes and populates chunks based on duration and ChunkDurationSeconds.
func (v *Video) CalculateChunks(framesPerSecond int) {
	duration := v.Metadata.DurationSeconds
	chunkDuration := constant.ChunkDurationSeconds
	totalChunks := int(math.Ceil(float64(duration) / float64(chunkDuration)))

	chunks := make([]Chunk, 0, totalChunks)
	for i := 0; i < totalChunks; i++ {
		startTime := i * chunkDuration
		endTime := (i + 1) * chunkDuration
		if endTime > duration {
			endTime = duration
		}

		chunk := Chunk{
			PartNumber:     i + 1,
			StartTime:      startTime,
			EndTime:        endTime,
			FramePerSecond: framesPerSecond,
			Status:         value_object.ChunkStatusPending,
			VideoObjectID:  fmt.Sprintf("videos/%s/chunks/part_%d.mp4", v.ID, i+1),
		}
		chunks = append(chunks, chunk)
	}

	v.Chunks = chunks
}

// UpdateChunkStatus updates the status of a specific chunk by PartNumber.
func (v *Video) UpdateChunkStatus(partNumber int, status value_object.ChunkStatus) error {
	for i := range v.Chunks {
		if v.Chunks[i].PartNumber == partNumber {
			v.Chunks[i].Status = status
			v.UpdatedAt = time.Now()
			return nil
		}
	}
	return exception.ErrChunkNotFound
}

// AllChunksProcessed checks if all chunks have status PROCESSED.
func (v *Video) AllChunksProcessed() bool {
	for _, chunk := range v.Chunks {
		if chunk.Status != value_object.ChunkStatusProcessed {
			return false
		}
	}
	return len(v.Chunks) > 0
}

// TransitionTo transitions the video status with state machine validation.
// Valid transitions: PENDING → PROCESSING, PROCESSING → COMPLETED, any → ERROR.
func (v *Video) TransitionTo(status value_object.VideoStatus) error {
	valid := false
	switch status {
	case value_object.VideoStatusProcessing:
		valid = v.Status == value_object.VideoStatusPending
	case value_object.VideoStatusCompleted:
		valid = v.Status == value_object.VideoStatusProcessing
	case value_object.VideoStatusError:
		valid = v.Status == value_object.VideoStatusPending || v.Status == value_object.VideoStatusProcessing
	}

	if !valid {
		return fmt.Errorf("%w: cannot transition from %s to %s", exception.ErrInvalidStatusTransition, v.Status, status)
	}

	v.Status = status
	v.UpdatedAt = time.Now()
	return nil
}

// Complete marks the video as COMPLETED and sets the DownloadURL.
func (v *Video) Complete(downloadURL string) error {
	if err := v.TransitionTo(value_object.VideoStatusCompleted); err != nil {
		return err
	}
	v.FileLocation.DownloadURL = downloadURL
	return nil
}

// Fail marks the video as ERROR and sets the cause.
func (v *Video) Fail(cause string) error {
	if err := v.TransitionTo(value_object.VideoStatusError); err != nil {
		return err
	}
	v.ErrorCause = cause
	return nil
}
