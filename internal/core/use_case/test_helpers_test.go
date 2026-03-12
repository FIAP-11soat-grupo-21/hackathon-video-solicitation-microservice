package use_case

import (
	"context"
	"fmt"
	"time"

	"video_solicitation_microservice/internal/common/pkg/identity"
	"video_solicitation_microservice/internal/core/domain/entity"
	"video_solicitation_microservice/internal/core/domain/value_object"
	"video_solicitation_microservice/internal/core/dto"
)

type videoRepoStub struct {
	saveFunc         func(context.Context, *entity.Video) error
	findByIDFunc     func(context.Context, string) (*entity.Video, error)
	findByUserIDFunc func(context.Context, string) ([]*entity.Video, error)
	updateFunc       func(context.Context, *entity.Video) error
	deleteFunc       func(context.Context, string, string) error

	savedVideo   *entity.Video
	updatedVideo *entity.Video
	deleteVideo  string
	deleteUser   string
	findCalls    int
}

func (stub *videoRepoStub) Save(ctx context.Context, video *entity.Video) error {
	stub.savedVideo = video
	if stub.saveFunc != nil {
		return stub.saveFunc(ctx, video)
	}
	return nil
}

func (stub *videoRepoStub) FindByID(ctx context.Context, id string) (*entity.Video, error) {
	stub.findCalls++
	if stub.findByIDFunc != nil {
		return stub.findByIDFunc(ctx, id)
	}
	return nil, nil
}

func (stub *videoRepoStub) FindByUserID(ctx context.Context, userID string) ([]*entity.Video, error) {
	if stub.findByUserIDFunc != nil {
		return stub.findByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (stub *videoRepoStub) Update(ctx context.Context, video *entity.Video) error {
	stub.updatedVideo = video
	if stub.updateFunc != nil {
		return stub.updateFunc(ctx, video)
	}
	return nil
}

func (stub *videoRepoStub) Delete(ctx context.Context, videoID string, userID string) error {
	stub.deleteVideo = videoID
	stub.deleteUser = userID
	if stub.deleteFunc != nil {
		return stub.deleteFunc(ctx, videoID, userID)
	}
	return nil
}

type uploadCall struct {
	bucket     string
	key        string
	expiration time.Duration
}

type downloadCall struct {
	bucket     string
	key        string
	expiration time.Duration
}

type blobStorageStub struct {
	uploadFunc   func(context.Context, string, string, time.Duration) (string, error)
	downloadFunc func(context.Context, string, string, time.Duration) (string, error)
	deleteFunc   func(context.Context, string, string) error

	uploadCalls   []uploadCall
	downloadCalls []downloadCall
	deleteBucket  string
	deletePrefix  string
}

func (stub *blobStorageStub) GeneratePreSignedUploadURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error) {
	stub.uploadCalls = append(stub.uploadCalls, uploadCall{bucket: bucket, key: key, expiration: expiration})
	if stub.uploadFunc != nil {
		return stub.uploadFunc(ctx, bucket, key, expiration)
	}
	return "", nil
}

func (stub *blobStorageStub) GeneratePreSignedDownloadURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error) {
	stub.downloadCalls = append(stub.downloadCalls, downloadCall{bucket: bucket, key: key, expiration: expiration})
	if stub.downloadFunc != nil {
		return stub.downloadFunc(ctx, bucket, key, expiration)
	}
	return "", nil
}

func (stub *blobStorageStub) DeleteObjectsByPrefix(ctx context.Context, bucket, prefix string) error {
	stub.deleteBucket = bucket
	stub.deletePrefix = prefix
	if stub.deleteFunc != nil {
		return stub.deleteFunc(ctx, bucket, prefix)
	}
	return nil
}

type publisherStub struct {
	allChunksProcessedFunc func(context.Context, dto.AllChunksProcessedEvent) error
	errorFunc              func(context.Context, dto.VideoProcessingErrorEvent) error

	allChunksProcessedEvents []dto.AllChunksProcessedEvent
	errorEvents              []dto.VideoProcessingErrorEvent
}

func (stub *publisherStub) PublishAllChunksProcessed(ctx context.Context, payload dto.AllChunksProcessedEvent) error {
	stub.allChunksProcessedEvents = append(stub.allChunksProcessedEvents, payload)
	if stub.allChunksProcessedFunc != nil {
		return stub.allChunksProcessedFunc(ctx, payload)
	}
	return nil
}

func (stub *publisherStub) PublishVideoProcessingError(ctx context.Context, payload dto.VideoProcessingErrorEvent) error {
	stub.errorEvents = append(stub.errorEvents, payload)
	if stub.errorFunc != nil {
		return stub.errorFunc(ctx, payload)
	}
	return nil
}

func validCreateVideoInput() dto.CreateVideoInput {
	return dto.CreateVideoInput{
		User: dto.UserDTO{
			ID:    identity.NewUUIDV7(),
			Name:  "Mateus",
			Email: "mateus@example.com",
		},
		Metadata: dto.MetadataDTO{
			FileName:        "video.mp4",
			DurationSeconds: 400,
			SizeBytes:       4096,
		},
		FramesPerSecond: 24,
	}
}

func validVideo(status value_object.VideoStatus) *entity.Video {
	id := identity.NewUUIDV7()
	return &entity.Video{
		ID: id,
		User: entity.User{
			ID:    identity.NewUUIDV7(),
			Name:  "Mateus",
			Email: "mateus@example.com",
		},
		Metadata: entity.Metadata{
			FileName:        "video.mp4",
			DurationSeconds: 400,
			SizeBytes:       4096,
		},
		Status: status,
		Chunks: []entity.Chunk{
			{PartNumber: 1, Status: value_object.ChunkStatusPending, VideoObjectID: fmt.Sprintf("videos/%s/chunks/part_1.mp4", id)},
			{PartNumber: 2, Status: value_object.ChunkStatusPending, VideoObjectID: fmt.Sprintf("videos/%s/chunks/part_2.mp4", id)},
		},
		FileLocation: entity.FileLocation{
			BucketName:       "bucket-test",
			VideoChunkFolder: fmt.Sprintf("videos/%s/chunks/", id),
			ImageFolder:      fmt.Sprintf("videos/%s/images/", id),
		},
		CreatedAt: time.Now().Add(-time.Minute),
		UpdatedAt: time.Now().Add(-time.Minute),
	}
}

func withStatus(input dto.UpdateVideoStatusInput, status string) dto.UpdateVideoStatusInput {
	input.Status = status
	return input
}
