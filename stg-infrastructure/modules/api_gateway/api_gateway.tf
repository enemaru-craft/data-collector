variable "routes" {
  type = list(object({
    method = string
    path   = string
  }))
  default = [
    { method = "GET", path = "/topic1" },
    { method = "GET", path = "/topic2" }
  ]
}



resource "aws_apigatewayv2_api" "get_iot_device_info_api" {
  name          = "mqtt_data_api"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.get_iot_device_info_api.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_apigatewayv2_route" "routes" {
  for_each = {
    for r in var.routes : "${r.method} ${r.path}" => r
  }

  api_id    = aws_apigatewayv2_api.get_iot_device_info_api.id
  route_key = "${each.value.method} ${each.value.path}"
  target    = "integrations/${aws_apigatewayv2_integration.lambda_integration.id}"
}

# API Gateway と Lambda の統合設定
resource "aws_apigatewayv2_integration" "lambda_integration" {
  api_id                 = aws_apigatewayv2_api.get_iot_device_info_api.id
  integration_type       = "AWS_PROXY"
  integration_uri        = var.management_world_data_lambda_arn
  payload_format_version = "2.0"
}


resource "aws_lambda_permission" "allow_api_gateway_invoke_lambda" {
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  function_name = var.management_world_data_lambda_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.get_iot_device_info_api.execution_arn}/*/*"
}
