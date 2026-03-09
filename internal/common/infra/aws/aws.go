package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type AWSClients struct {
	S3Client  *s3.Client
	SQSClient *sqs.Client
	SNSClient *sns.Client
}

func NewAWSClients(ctx context.Context, region, endpoint string) (*AWSClients, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, reg string, options ...interface{}) (aws.Endpoint, error) {
			if endpoint != "" {
				return aws.Endpoint{
					URL:               endpoint,
					HostnameImmutable: true,
					PartitionID:       "aws",
				}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		},
	)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "test")),
	)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	sqsClient := sqs.NewFromConfig(cfg)
	snsClient := sns.NewFromConfig(cfg)

	return &AWSClients{
		S3Client:  s3Client,
		SQSClient: sqsClient,
		SNSClient: snsClient,
	}, nil
}
