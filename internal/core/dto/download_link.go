package dto

type DownloadLinkOutput struct {
	VideoID     string `json:"video_id"`
	Status      string `json:"status"`
	DownloadURL string `json:"download_url"`
}
