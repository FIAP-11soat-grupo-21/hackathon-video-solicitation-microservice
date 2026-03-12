package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	blobStorageAdapter "video_solicitation_microservice/internal/adapter/driven/blob_storage"
	databaseAdapter "video_solicitation_microservice/internal/adapter/driven/database"
	messagingAdapter "video_solicitation_microservice/internal/adapter/driven/messaging"
	apiAdapter "video_solicitation_microservice/internal/adapter/driver/api"
	messagingDriver "video_solicitation_microservice/internal/adapter/driver/messaging"
	"video_solicitation_microservice/internal/common/config/env"
	infraAPI "video_solicitation_microservice/internal/common/infra/api"
	infraAWS "video_solicitation_microservice/internal/common/infra/aws"
	infraDB "video_solicitation_microservice/internal/common/infra/database"
	"video_solicitation_microservice/internal/core/use_case"
)

func main() {
	fmt.Println("INICIO MAIN")
	ctx := context.Background()

	// Load configuration
	cfg := env.LoadConfig()
	log.Printf(" ✅ Starting video-solicitation service on %s:%s", cfg.API.Host, cfg.API.Port)
	log.Printf("AWS_ENDPOINT: %s", cfg.AWS.Endpoint)
	log.Printf("AWS_ENDPOINT_DYNAMO: %s", cfg.AWS.EndpointDynamo)
	// Initialize AWS config for DynamoDB
	configOptions := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.AWS.Region),
	}

	if cfg.AWS.EndpointDynamo != "" {
		configOptions = append(configOptions, config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{URL: cfg.AWS.EndpointDynamo}, nil
			},
		)))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, configOptions...)

	// Initialize DynamoDB client
	db, err := infraDB.NewDynamoDB(awsCfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	// DynamoDB client does not require Close, so omit defer db.Close()

	fmt.Println("AWS_ENDPOINT:", cfg.AWS.Endpoint)
	fmt.Println("AWS_ENDPOINT_DYNAMO:", cfg.AWS.EndpointDynamo)
	fmt.Println("[DEBUG] Antes de rodar migration: AWS_ENDPOINT=", cfg.AWS.Endpoint, ", AWS_ENDPOINT_DYNAMO=", cfg.AWS.EndpointDynamo)
	if cfg.DB.RunMigrate {
		fmt.Println("[DEBUG] Chamando RunMigrationsDynamoDB...")
		if err := infraDB.RunMigrationsDynamoDB(db, ctx); err != nil {
			fmt.Println("Failed to run DynamoDB migrations:", err)
			log.Fatalf("Failed to run DynamoDB migrations: %v", err)
		}
		fmt.Println("[DEBUG] RunMigrationsDynamoDB finalizada")
		fmt.Println("DynamoDB migrations completed")
	}

	// Initialize AWS clients
	awsClients, err := infraAWS.NewAWSClients(ctx, cfg.AWS.Region, cfg.AWS.Endpoint)
	if err != nil {
		log.Fatalf("Failed to initialize AWS clients: %v", err)
	}

	// Instantiate driven adapters
	videoRepo := databaseAdapter.NewVideoRepositoryDynamoDB(db)
	blobStore := blobStorageAdapter.NewS3Storage(awsClients.S3Client)
	publisher := messagingAdapter.NewSNSPublisher(awsClients.SNSClient, cfg.AWS.SNS.AllChunkProcessedEventARN)

	// Instantiate use cases
	createVideoUC := use_case.NewCreateVideo(videoRepo, blobStore)
	getDownloadUC := use_case.NewGetDownloadLink(videoRepo, blobStore)
	getVideosByUserUC := use_case.NewGetVideosByUser(videoRepo)
	updateChunkUC := use_case.NewUpdateChunkStatus(videoRepo, publisher)
	updateVideoUC := use_case.NewUpdateVideoStatus(videoRepo)

	// Instantiate driver adapters
	httpHandler := apiAdapter.NewVideoHandler(createVideoUC, getDownloadUC, getVideosByUserUC, updateVideoUC, updateChunkUC)
	chunkConsumer := messagingDriver.NewChunkStatusConsumer(awsClients.SQSClient, cfg.AWS.SQS.UpdateChunkStatusQueueURL, updateChunkUC)
	videoConsumer := messagingDriver.NewVideoStatusConsumer(awsClients.SQSClient, cfg.AWS.SQS.UpdateVideoStatusQueueURL, updateVideoUC)

	// Setup HTTP router and register routesVIDEO_PROCESSED_ERROR_EVENT_ARN
	router := infraAPI.NewRouter()
	httpHandler.RegisterRoutes(router.Group("/v1"))

	// Start SQS consumers in goroutines
	consumerCtx, cancelConsumers := context.WithCancel(ctx)
	defer cancelConsumers()

	go chunkConsumer.Start(consumerCtx)
	go videoConsumer.Start(consumerCtx)
	log.Println("SQS consumers started")

	// Start HTTP server (blocks until shutdown signal)
	infraAPI.StartServer(ctx, router, cfg.API.Host, cfg.API.Port)

	// Cancel consumer context on shutdown
	cancelConsumers()
	log.Println("Service stopped")
}
