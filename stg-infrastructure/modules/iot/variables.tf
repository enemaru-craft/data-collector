variable "stg_power_data_registration_lambda_arn" {
  type        = string
  description = "Stgにおいて発電量データを登録するLambdaのARN"
}

variable "stg_power_data_registration_lambda_function_name" {
  type        = string
  description = "Stgにおいて発電量データを登録するLambdaの関数名"
}
