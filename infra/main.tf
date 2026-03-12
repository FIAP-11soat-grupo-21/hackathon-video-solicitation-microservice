module "video_solicitation_api" {
  source     = "git::ssh://git@github.com/FIAP-11soat-grupo-21/infra-core.git//modules/ECS-Service?ref=main"
  depends_on = [aws_lb_listener.listener]

  cluster_id            = data.terraform_remote_state.ecs.outputs.cluster_id
  ecs_security_group_id = data.terraform_remote_state.ecs.outputs.ecs_security_group_id

  cloudwatch_log_group     = data.terraform_remote_state.ecs.outputs.cloudwatch_log_group
  ecs_container_image      = var.image_name
  ecs_container_name       = var.application_name
  ecs_container_port       = var.image_port
  ecs_service_name         = var.application_name
  ecs_desired_count        = var.desired_count
  registry_credentials_arn = data.terraform_remote_state.ghcr_secret.outputs.secret_arn

  ecs_container_environment_variables = merge(var.container_environment_variables,
    {
      AWS_S3_BUCKET_NAME : data.terraform_remote_state.s3.outputs.bucket_name
      SQS_UPDATE_VIDEO_CHUNK_STATUS_QUEUE_URL : data.terraform_remote_state.sqs_update_video_chunk_status.outputs.sqs_queue_url
      SQS_UPDATE_VIDEO_STATUS_QUEUE_URL : data.terraform_remote_state.sqs_update_video_status.outputs.sqs_queue_url
      SQS_VIDEO_PROCESSING_ERROR_QUEUE_URL : data.terraform_remote_state.sqs_update_video_status.outputs.sqs_queue_url
      ALL_CHUNK_PROCESSED_EVENT_ARN : data.terraform_remote_state.sns_all_chunks_processed.outputs.topic_arn
      VIDEO_PROCESSED_ERROR_EVENT_ARN : data.terraform_remote_state.sns_video_processed_error.outputs.topic_arn
    }
  )
  
  // todo: adicionar secrets do DynamoDB
  ecs_container_secrets = merge(
    var.container_secrets,
    {
    }
  )

  private_subnet_ids      = data.terraform_remote_state.network_vpc.outputs.private_subnets
  task_execution_role_arn = data.terraform_remote_state.ecs.outputs.task_execution_role_arn
  task_role_policy_arns   = var.task_role_policy_arns
  alb_target_group_arn    = aws_alb_target_group.target_group.arn
  alb_security_group_id   = data.terraform_remote_state.alb.outputs.alb_security_group_id

  project_common_tags = data.terraform_remote_state.app_registry.outputs.app_registry_application_tag
}

module "GetVideoSolicitationAPIRoute" {
  source     = "git::ssh://git@github.com/FIAP-11soat-grupo-21/infra-core.git//modules/API-Gateway-Routes?ref=main"
  depends_on = [module.video_solicitation_api]

  api_id       = data.terraform_remote_state.api_gateway.outputs.api_id
  alb_proxy_id = aws_apigatewayv2_integration.alb_proxy.id

  endpoints = {
    download_video_zip = {
      route_key           = "GET /videos/{id}/download"
      restricted          = false
      auth_integration_id = data.terraform_remote_state.auth.outputs.auth_id
    },
    create_video = {
      route_key           = "POST /videos"
      restricted          = true
      auth_integration_id = data.terraform_remote_state.auth.outputs.auth_id
    }
  }
}

module "chunk_upload_notifier_lambda" {
  source = "git::https://github.com/FIAP-11soat-grupo-21/infra-core.git//modules/Lambda?ref=main"

  lambda_name = "chunk-upload-notifier"
  handler     = "bootstrap"
  runtime     = "provided.al2023"
  subnet_ids  = data.terraform_remote_state.network_vpc.outputs.private_subnets
  
  environment = merge(
    var.lambda_environment_variables,
    {
      SNS_CHUNK_UPLOADED_TOPIC = data.terraform_remote_state.sns_chunk_uploaded.outputs.topic_arn
      DYNAMODB_TABLE_NAME      = var.dynamodb_table_name
    }
  )
  
  vpc_id      = data.terraform_remote_state.network_vpc.outputs.vpc_id
  memory_size = 256
  timeout     = 30

  s3_bucket = var.lambda_bucket_name
  s3_key    = var.chunk_upload_notifier_s3_key

  role_permissions = {
    dynamodb = {
      actions = [
        "dynamodb:GetItem"
      ]
      resources = [
        "arn:aws:dynamodb:${data.aws_region.current.id}:*:table/${var.dynamodb_table_name}"
      ]
    }
    sqs = {
      actions = [
        "sqs:ReceiveMessage",
        "sqs:DeleteMessage",
        "sqs:GetQueueAttributes",
        "sqs:GetQueueUrl"
      ]
      resources = [
        data.terraform_remote_state.sqs_chunk_processor.outputs.sqs_queue_arn
      ]
    }
    sns = {
      actions = [
        "sns:Publish"
      ]
      resources = [
        data.terraform_remote_state.sns_chunk_uploaded.outputs.topic_arn
      ]
    }
  }

  tags = data.terraform_remote_state.app_registry.outputs.app_registry_application_tag
}

resource "aws_lambda_event_source_mapping" "chunk_upload_notifier" {
  event_source_arn = data.terraform_remote_state.sqs_chunk_processor.outputs.sqs_queue_arn
  function_name    = module.chunk_upload_notifier_lambda.lambda_arn
  batch_size       = 1
}
