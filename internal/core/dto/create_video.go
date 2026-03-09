package dto

// CreateVideoInput is the input for POST /videos
type CreateVideoInput struct {
	User            UserDTO     `json:"user"`
	Metadata        MetadataDTO `json:"metadata"`
	FramesPerSecond int         `json:"frames_per_second"`
}

type UserDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type MetadataDTO struct {
	FileName        string `json:"file_name"`
	DurationSeconds int    `json:"duration_seconds"`
	SizeBytes       int64  `json:"size_bytes"`
}

// CreateVideoOutput is the response for POST /videos
type CreateVideoOutput struct {
	VideoID string           `json:"video_id"`
	Status  string           `json:"status"`
	Chunks  []ChunkOutputDTO `json:"chunks"`
}

type ChunkOutputDTO struct {
	PartNumber int    `json:"part_number"`
	StartTime  int    `json:"start_time"`
	EndTime    int    `json:"end_time"`
	UploadURL  string `json:"upload_url"`
}
