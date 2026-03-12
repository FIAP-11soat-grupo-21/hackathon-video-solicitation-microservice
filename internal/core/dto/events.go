package dto

type AllChunksProcessedEvent struct {
	VideoID     string  `json:"video_id"`
	User        UserDTO `json:"user"`
	BucketName  string  `json:"bucket_name"`
	ImageFolder string  `json:"image_folder"`
}

type VideoProcessingErrorEvent struct {
	VideoID       string  `json:"video_id"`
	User          UserDTO `json:"user"`
	Status        string  `json:"status"`
	Cause         string  `json:"cause"`
	SystemTrigger string  `json:"system_trigger"`
}
