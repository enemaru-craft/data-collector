module "mqtt_lambda" {
  source = "./modules/lambda"
}

module "mqtt_iot" {
  source = "./modules/iot"

  # IoT Coreのデータ加工用のLambda
  lambda_arn    = module.mqtt_lambda.lambda_arn
  function_name = module.mqtt_lambda.function_name
}

module "dynamodb" {
  source = "./modules/dynamodb"
}
