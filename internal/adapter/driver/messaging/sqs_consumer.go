package messaging

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSConsumer struct {
	client   *sqs.Client
	queueURL string
	handler  func(ctx context.Context, msg []byte) error
	name     string
}

func NewSQSConsumer(client *sqs.Client, queueURL string, name string, handler func(ctx context.Context, msg []byte) error) *SQSConsumer {
	return &SQSConsumer{
		client:   client,
		queueURL: queueURL,
		handler:  handler,
		name:     name,
	}
}

func (c *SQSConsumer) Start(ctx context.Context) {
	log.Printf("[%s] SQS consumer started, waiting for queue to be available: %s", c.name, c.queueURL)

	c.waitForConnection(ctx)

	log.Printf("[%s] SQS consumer connected, polling queue: %s", c.name, c.queueURL)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] SQS consumer stopping...", c.name)
			return
		default:
			c.pollMessages(ctx)
		}
	}
}

func (c *SQSConsumer) waitForConnection(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, err := c.client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
				QueueUrl:       aws.String(c.queueURL),
				AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
			})
			if err == nil {
				return
			}
			log.Printf("[%s] Queue not available yet, retrying in 5s... (%v)", c.name, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}
}

func (c *SQSConsumer) pollMessages(ctx context.Context) {
	output, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.queueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20, // long polling
		VisibilityTimeout:   30,
	})
	if err != nil {
		if ctx.Err() != nil {
			return // context cancelled, shutting down
		}
		log.Printf("[%s] Error receiving messages: %v", c.name, err)
		time.Sleep(5 * time.Second)
		return
	}

	for _, msg := range output.Messages {
		c.processMessage(ctx, msg)
	}
}

func (c *SQSConsumer) processMessage(ctx context.Context, msg sqstypes.Message) {
	if msg.Body == nil {
		return
	}

	body := extractMessageBody(*msg.Body)

	if err := c.handler(ctx, []byte(body)); err != nil {
		log.Printf("[%s] Error processing message %s: %v", c.name, *msg.MessageId, err)
		return // message will return to queue after visibility timeout
	}

	// Delete message on success
	_, err := c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Printf("[%s] Error deleting message %s: %v", c.name, *msg.MessageId, err)
	}
}

// extractMessageBody handles SNS-wrapped messages.
// When a message comes through SNS→SQS subscription, the actual payload
// is nested inside the SNS envelope's "Message" field.
func extractMessageBody(body string) string {
	// For now, return the body as-is.
	// If messages come via SNS subscription, they'll be in SNS envelope format.
	// Direct SQS messages are plain JSON.
	return body
}
