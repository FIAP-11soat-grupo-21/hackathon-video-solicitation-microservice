package dto

type ChunkUploadedDTO struct {
	Bucket           string  `json:"bucket"`
	VideoObjectID    string  `json:"video_object_id"`
	User             UserDTO `json:"user"`
	ImageDestination string  `json:"image_destination"`
	FramePerSecond   int     `json:"frame_per_second"`
	ChunkPart        int     `json:"chunk_part"`
}

type AllChunksProcessedEvent struct {
	VideoID     string  `json:"video_id"`
	User        UserDTO `json:"user"`
	BucketName  string  `json:"bucket_name"`
	ImageFolder string  `json:"image_folder"`
}
