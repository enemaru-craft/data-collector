variable "stg_device_and_world_management_routes" {
  type = list(object({
    method = string
    path   = string
  }))
  default = [
    { method = "POST", path = "/register-new-power-generation-module", description = "新しい発電モジュールを登録する" },
  ]
}



resource "aws_apigatewayv2_api" "stg_device_and_world_management_api" {
  name          = "stg_device_and_world_management_api"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.stg_device_and_world_management_api.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_apigatewayv2_route" "stg_device_and_world_management_routes" {
  for_each = {
    for r in var.stg_device_and_world_management_routes : "${r.method} ${r.path}" => r
  }

  api_id    = aws_apigatewayv2_api.stg_device_and_world_management_api.id
  route_key = "${each.value.method} ${each.value.path}"
  target    = "integrations/${aws_apigatewayv2_integration.stg_integrate_management_api_and_lambda.id}"
}

# API Gateway と Lambda の統合設定
resource "aws_apigatewayv2_integration" "stg_integrate_management_api_and_lambda" {
  api_id                 = aws_apigatewayv2_api.stg_device_and_world_management_api.id
  integration_type       = "AWS_PROXY"
  integration_uri        = var.stg_management_device_and_world_data_lambda_function_arn
  payload_format_version = "2.0"
}


resource "aws_lambda_permission" "stg_grant_calling_lambda_permission_to_management_api" {
  statement_id  = "GrantCallingLambdaPermissionToManagementApi"
  action        = "lambda:InvokeFunction"
  function_name = var.stg_management_device_and_world_data_lambda_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.stg_device_and_world_management_api.execution_arn}/*/*"
}
