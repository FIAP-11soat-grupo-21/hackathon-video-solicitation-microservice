package env

import (
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

var (
	instance *Config
	once     sync.Once
)

type Config struct {
	GoEnv string
	API   struct {
		Port string
		Host string
	}
	DB struct {
		Path       string
		RunMigrate bool
	}
	AWS struct {
		Region   string
		Endpoint string
		S3       struct {
			BucketName string
		}
		SQS struct {
			UpdateChunkStatusQueueURL    string
			UpdateVideoStatusQueueURL    string
			VideoProcessingErrorQueueURL string
		}
		SNS struct {
			AllChunkProcessedEventARN   string
			VideoProcessedErrorEventARN string
		}
	}
}

func LoadConfig() *Config {
	once.Do(func() {
		instance = new(Config)
		instance.Load()
	})
	return instance
}

func (c *Config) Load() {
	dotEnvPath := ".env"
	_, err := os.Stat(dotEnvPath)

	if err == nil {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	// Environment
	c.GoEnv = getEnv("GO_ENV", "development")

	// API
	c.API.Port = getEnv("API_PORT", "8080")
	c.API.Host = getEnv("API_HOST", "0.0.0.0")

	// Database
	c.DB.Path = getEnv("DB_PATH", "./data/video_solicitation.db")
	c.DB.RunMigrate = getEnvBool("DB_RUN_MIGRATE", true)

	// AWS
	c.AWS.Region = getEnv("AWS_REGION", "us-east-2")
	c.AWS.Endpoint = getEnv("AWS_ENDPOINT", "")

	// S3
	c.AWS.S3.BucketName = getEnv("AWS_S3_BUCKET_NAME", "video-solicitation-bucket")

	// SQS
	c.AWS.SQS.UpdateChunkStatusQueueURL = getEnv("SQS_UPDATE_VIDEO_CHUNK_STATUS_QUEUE_URL", "")
	c.AWS.SQS.UpdateVideoStatusQueueURL = getEnv("SQS_UPDATE_VIDEO_STATUS_QUEUE_URL", "")
	c.AWS.SQS.VideoProcessingErrorQueueURL = getEnv("SQS_VIDEO_PROCESSING_ERROR_QUEUE_URL", "")

	// SNS
	c.AWS.SNS.AllChunkProcessedEventARN = getEnv("ALL_CHUCK_PROCESSED_EVENT_ARN", "")
	c.AWS.SNS.VideoProcessedErrorEventARN = getEnv("VIDEO_PROCESSED_ERROR_EVENT_ARN", "")
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return b
}

func (c *Config) IsProduction() bool {
	return c.GoEnv == "production"
}

func (c *Config) IsDevelopment() bool {
	return c.GoEnv == "development"
}
