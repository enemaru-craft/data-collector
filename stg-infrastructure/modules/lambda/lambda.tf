resource "aws_lambda_function" "mqtt_data_controller" {
  function_name = "mqtt_data_controller"
  runtime       = "provided.al2023"
  handler       = "main"

  role = aws_iam_role.lambda_exec.arn

  filename = "${path.module}/lambda_function.zip"

  lifecycle {
    ignore_changes = [
      filename,
      source_code_hash,
      s3_bucket,
      s3_key,
    ]
  }
}


resource "aws_lambda_function" "management_world_data_lambda" {
  function_name = "management_world_data_lambda"
  runtime       = "provided.al2023"
  handler       = "main"

  role = aws_iam_role.lambda_exec.arn

  filename = "${path.module}/lambda_function.zip"

  lifecycle {
    ignore_changes = [
      filename,
      source_code_hash,
      s3_bucket,
      s3_key,
    ]
  }
}
