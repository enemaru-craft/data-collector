# IoT Coreデータ加工用のLambda関数のARNと名前を出力する

output "stg_power_data_registration_lambda_arn" {
  value       = aws_lambda_function.stg_power_data_registration_lambda.arn
  description = "Stgにおいて発電量データを登録するLambdaのARN"
}

output "stg_power_data_registration_lambda_function_name" {
  value       = aws_lambda_function.stg_power_data_registration_lambda.function_name
  description = "Stgにおいて発電量データを登録するLambdaの関数名"
}

output "stg_management_device_and_world_data_lambda_function_arn" {
  value       = aws_lambda_function.stg_management_device_and_world_data_lambda.arn
  description = "Stgにおいてデバイスとマイクラワールドデータを管理するLambdaのARN"
}

output "stg_management_device_and_world_data_lambda_function_name" {
  value       = aws_lambda_function.stg_management_device_and_world_data_lambda.function_name
  description = "Stgにおいてデバイスとマイクラワールドデータを管理するLambdaの関数名"
}
