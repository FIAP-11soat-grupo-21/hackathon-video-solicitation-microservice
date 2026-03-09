package dto

type UpdateChunkStatusInput struct {
	VideoID   string  `json:"video_id"`
	User      UserDTO `json:"user"`
	ChunkPart int     `json:"chunk_part"`
	Status    string  `json:"status"`
}
