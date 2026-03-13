resource "aws_dynamodb_table" "videos" {
  name           = "videos"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "user_id"
  range_key      = "video_id"

  attribute {
    name = "user_id"
    type = "S"
  }
  attribute {
    name = "video_id"
    type = "S"
  }
  attribute {
    name = "status"
    type = "S"
  }
  attribute {
    name = "created_at"
    type = "S"
  }

  global_secondary_index {
    name            = "video_id-index"
    hash_key        = "video_id"
    projection_type = "ALL"
  }

  global_secondary_index {
    name            = "status-index"
    hash_key        = "status"
    range_key       = "created_at"
    projection_type = "ALL"
  }
}

resource "aws_dynamodb_table" "chunks" {
  name           = "chunks"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "video_id"
  range_key      = "part_number"

  attribute {
    name = "video_id"
    type = "S"
  }
  attribute {
    name = "part_number"
    type = "N"
  }

  global_secondary_index {
    name            = "video_id-index"
    hash_key        = "video_id"
    range_key       = "part_number"
    projection_type = "ALL"
  }
}