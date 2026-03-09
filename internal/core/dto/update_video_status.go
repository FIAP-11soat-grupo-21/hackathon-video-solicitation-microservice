package dto

type UpdateVideoStatusInput struct {
	VideoID       string  `json:"video_id"`
	User          UserDTO `json:"user"`
	Status        string  `json:"status"`
	DownloadURL   string  `json:"download_url,omitempty"`
	Cause         string  `json:"cause,omitempty"`
	SystemTrigger string  `json:"system_trigger,omitempty"`
}
