# 3. IoTトピックルールでLambdaを呼び出す設定
resource "aws_iot_topic_rule" "mqtt_to_lambda" {
  name        = "mqtt_to_lambda_rule"
  description = "Invoke Lambda on MQTT messages"
  sql         = "SELECT * FROM 'mqtt_test/test'"
  sql_version = "2016-03-23"
  enabled     = true

  lambda {
    function_arn = aws_lambda_function.mqtt_data_controller.arn
  }

}

resource "aws_lambda_permission" "allow_iot" {
  statement_id  = "AllowExecutionFromIoT"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.mqtt_data_controller.function_name
  principal     = "iot.amazonaws.com"
  source_arn    = aws_iot_topic_rule.mqtt_to_lambda.arn
}
