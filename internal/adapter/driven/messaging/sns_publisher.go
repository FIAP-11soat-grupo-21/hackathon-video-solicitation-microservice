package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"video_solicitation_microservice/internal/core/domain/port"
	"video_solicitation_microservice/internal/core/dto"
)

type snsPublisher struct {
	client   *sns.Client
	topicARN string
}

func NewSNSPublisher(client *sns.Client, topicARN string) port.MessagePublisher {
	return &snsPublisher{
		client:   client,
		topicARN: topicARN,
	}
}

func (p *snsPublisher) PublishAllChunksProcessed(ctx context.Context, payload dto.AllChunksProcessedEvent) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = p.client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.topicARN),
		Message:  aws.String(string(data)),
	})
	if err != nil {
		return fmt.Errorf("failed to publish to SNS: %w", err)
	}

	return nil
}
