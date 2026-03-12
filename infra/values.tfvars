application_name = "video-solicitation-api"
image_name       = "GHCR_IMAGE_TAG"
image_port       = 8085
app_path_pattern = ["/videos*", "/videos/*"]

# =======================================================
# Configurações do ECS Service
# =======================================================
container_environment_variables = {
  GO_ENV : "production"
  API_PORT : "8085"
  API_HOST : "0.0.0.0"

  AWS_REGION : "us-east-2"

  DB_RUN_MIGRATIONS : "true"
}

container_secrets = {}
health_check_path = "/health"
task_role_policy_arns = [
  "arn:aws:iam::aws:policy/AmazonS3FullAccess",
  "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess",
  "arn:aws:iam::aws:policy/AmazonSQSFullAccess",
  "arn:aws:iam::aws:policy/AmazonSNSFullAccess",
]
alb_is_internal = true

# =======================================================
# Configurações do API Gateaway
# =======================================================
# API Gateway
apigw_integration_type       = "HTTP_PROXY"
apigw_integration_method     = "ANY"
apigw_payload_format_version = "1.0"
apigw_connection_type        = "VPC_LINK"

authorization_name = "CognitoAuthorizer"


# =======================================================
# Configurações da Lambda
# =======================================================
lambda_environment_variables = {}

lambda_bucket_name           = "fiap-hackathon-lambda-content-44573"
chunk_upload_notifier_s3_key = "chunk-upload-notifier.zip"
dynamodb_table_name          = "videos-05"
