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

  management_world_data_lambda_arn           = module.lambda.management_world_data_lambda_arn
  management_world_data_lambda_function_name = module.lambda.management_world_data_lambda_function_name
}
