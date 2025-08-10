# IoT Coreデータ加工用のLambda関数のARNと名前を出力する

output "lambda_arn" {
  value       = aws_lambda_function.mqtt_data_controller.arn
  description = "IoT Coreのデータ加工用LambdaのARN"
}

output "function_name" {
  value       = aws_lambda_function.mqtt_data_controller.function_name
  description = "IoT Coreのデータ加工用Lambdaの関数名"
}

output "management_world_data_lambda_arn" {
  value       = aws_lambda_function.management_world_data_lambda.arn
  description = "Management World Data LambdaのARN"
}

output "management_world_data_lambda_function_name" {
  value       = aws_lambda_function.management_world_data_lambda.function_name
  description = "Management World Data Lambdaの関数名"
}
