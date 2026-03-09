package entity

type FileLocation struct {
	BucketName       string
	VideoChunkFolder string // "videos/{video_id}/chunks/"
	ImageFolder      string // "videos/{video_id}/images/"
	DownloadURL      string // populated when status == COMPLETED
}
