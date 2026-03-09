package constant

import "time"

const (
	S3BucketName              = "fiapx-videos"
	SQSUpdateChunkStatusQueue = "update-video-chunk-status"
	SQSUpdateVideoStatusQueue = "update-video-status"
	SNSAllChunkProcessedTopic = "all-chunk-processed"
	ChunkDurationSeconds      = 180 // 3 minutes per chunk
	PreSignedURLExpiration    = 30 * time.Minute
)
