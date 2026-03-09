package main

import (
	"context"
	"log"

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
	ctx := context.Background()

	// Load configuration
	cfg := env.LoadConfig()
	log.Printf("Starting video-solicitation service on %s:%s", cfg.API.Host, cfg.API.Port)

	// Initialize SQLite database
	db, err := infraDB.NewSQLiteDB(cfg.DB.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if cfg.DB.RunMigrate {
		if err := infraDB.RunMigrations(db); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Database migrations completed")
	}

	// Initialize AWS clients
	awsClients, err := infraAWS.NewAWSClients(ctx, cfg.AWS.Region, cfg.AWS.Endpoint)
	if err != nil {
		log.Fatalf("Failed to initialize AWS clients: %v", err)
	}

	// Instantiate driven adapters
	videoRepo := databaseAdapter.NewVideoRepositorySQLite(db)
	blobStore := blobStorageAdapter.NewS3Storage(awsClients.S3Client)
	publisher := messagingAdapter.NewSNSPublisher(awsClients.SNSClient, cfg.AWS.SNS.AllChunkProcessedEventARN)

	// Instantiate use cases
	createVideoUC := use_case.NewCreateVideo(videoRepo, blobStore)
	getDownloadUC := use_case.NewGetDownloadLink(videoRepo, blobStore)
	updateChunkUC := use_case.NewUpdateChunkStatus(videoRepo, publisher)
	updateVideoUC := use_case.NewUpdateVideoStatus(videoRepo)

	// Instantiate driver adapters
	httpHandler := apiAdapter.NewVideoHandler(createVideoUC, getDownloadUC)
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
