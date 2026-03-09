package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"video_solicitation_microservice/internal/core/dto"
	"video_solicitation_microservice/internal/core/use_case"
)

func NewVideoStatusConsumer(sqsClient *sqs.Client, queueURL string, useCase *use_case.UpdateVideoStatus) *SQSConsumer {
	handler := func(ctx context.Context, msg []byte) error {
		var input dto.UpdateVideoStatusInput
		if err := json.Unmarshal(msg, &input); err != nil {
			return fmt.Errorf("failed to unmarshal video status message: %w", err)
		}
		return useCase.Execute(ctx, input)
	}

	return NewSQSConsumer(sqsClient, queueURL, "video-status-consumer", handler)
}
