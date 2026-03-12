package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"

	lambdaHandler "video_solicitation_microservice/internal/adapter/driver/lambda"
)

func main() {
	handler, err := lambdaHandler.NewChunkUploadNotifierHandler()
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	lambda.Start(handler.Handle)
}
