module "lambda" {
  source = "./modules/lambda"
}

module "mqtt_iot" {
  source = "./modules/iot"

  # IoT Coreのデータ加工用のLambda
  lambda_arn    = module.lambda.lambda_arn
  function_name = module.lambda.function_name
}

module "dynamodb" {
  source = "./modules/dynamodb"
}


module "api_gateway" {
  source = "./modules/api_gateway"

  stg_management_device_and_world_data_lambda_function_arn  = module.lambda.stg_management_device_and_world_data_lambda_function_arn
  stg_management_device_and_world_data_lambda_function_name = module.lambda.stg_management_device_and_world_data_lambda_function_name
}
