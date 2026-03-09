package entity

import "video_solicitation_microservice/internal/core/domain/value_object"

type Chunk struct {
	PartNumber     int
	StartTime      int
	EndTime        int
	FramePerSecond int
	Status         value_object.ChunkStatus
	VideoObjectID  string // "videos/{video_id}/chunks/part_1.mp4"
}
