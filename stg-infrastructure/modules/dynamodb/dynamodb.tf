resource "aws_dynamodb_table" "tmp_table" {
  name         = "tmp_table"
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "device_id"

  attribute {
    name = "device_id"
    type = "S"
  }
}
