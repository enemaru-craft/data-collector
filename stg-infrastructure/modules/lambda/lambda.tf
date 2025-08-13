resource "aws_lambda_function" "stg_power_data_registration_lambda" {
  function_name = "stg_power_data_registration_lambda"
  runtime       = "provided.al2023"
  handler       = "main"

  role = aws_iam_role.stg_lambda_exec_role.arn

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


resource "aws_lambda_function" "stg_management_device_and_world_data_lambda" {
  function_name = "stg_management_device_and_world_data_lambda"
  runtime       = "provided.al2023"
  handler       = "main"

  role = aws_iam_role.stg_lambda_exec_role.arn

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
