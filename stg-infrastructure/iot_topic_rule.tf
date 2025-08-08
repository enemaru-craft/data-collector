
# 変数を用いて複数のトピックを再帰的に実装する
variable "topics" {
  default = {
    test_topic_1 = "test/topic/1"
    test_topic_2 = "test/topic/2"
  }
}

resource "aws_iot_topic_rule" "mqtt_to_lambda" {
  for_each = var.topics

  name        = "mqtt_to_lambda_rule_${each.key}"
  description = "Invoke Lambda on MQTT messages"
  sql         = "SELECT * ,topic() as topic FROM '${each.value}'"
  sql_version = "2016-03-23"
  enabled     = true

  lambda {
    function_arn = aws_lambda_function.mqtt_data_controller.arn
  }

}

resource "aws_lambda_permission" "allow_iot" {
  for_each = var.topics

  statement_id  = "AllowExecutionFromIoT_${each.key}"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.mqtt_data_controller.function_name
  principal     = "iot.amazonaws.com"
  source_arn    = aws_iot_topic_rule.mqtt_to_lambda[each.key].arn
}
