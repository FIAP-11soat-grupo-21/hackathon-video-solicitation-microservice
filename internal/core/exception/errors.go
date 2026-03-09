package exception

import "errors"

var (
	ErrVideoNotFound           = errors.New("video not found")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrChunkNotFound           = errors.New("chunk not found")
	ErrVideoNotCompleted       = errors.New("video is not completed yet")
	ErrInvalidInput            = errors.New("invalid input data")
)
