package value_object

type VideoStatus string

const (
	VideoStatusPending    VideoStatus = "PENDING"
	VideoStatusProcessing VideoStatus = "PROCESSING"
	VideoStatusCompleted  VideoStatus = "COMPLETED"
	VideoStatusError      VideoStatus = "ERROR"
)

type ChunkStatus string

const (
	ChunkStatusPending    ChunkStatus = "PENDING"
	ChunkStatusProcessing ChunkStatus = "PROCESSING"
	ChunkStatusProcessed  ChunkStatus = "PROCESSED"
	ChunkStatusError      ChunkStatus = "ERROR"
)
