resource "aws_dynamodb_table" "mqtt_test_table" {
  name         = "mqtt_test_table"
  billing_mode = "PAY_PER_REQUEST"

  hash_key  = "client_id"
  range_key = "timestamp"

  attribute {
    name = "client_id"
    type = "S"
  }

  attribute {
    name = "timestamp"
    type = "N"
  }
}
