resource "aws_dynamodb_table" "tmp_table" {
  name         = "tmp_table"
  billing_mode = "PAY_PER_REQUEST"

  hash_key  = "session_id"
  range_key = "device_id"

  attribute {
    name = "session_id"
    type = "S"
  }
  attribute {
    name = "device_id"
    type = "S"
  }
}
