package constant

import "time"

const (
	ChunkDurationSeconds   = 180 // 3 minutes per chunk
	PreSignedURLExpiration = 30 * time.Minute
	SystemTriggerName      = "video_solicitation_service"
)
