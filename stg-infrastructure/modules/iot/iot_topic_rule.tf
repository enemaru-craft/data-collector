
# 変数を用いて複数のトピックを再帰的に実装する
variable "topics" {
  default = {
    register_geothermal = "register/geothermal"
    register_solar      = "register/solar"
    register_wind       = "register/wind"
    register_hydrogen   = "register/hydrogen"
    register_hand_crank = "register/hand-crank"
  }
}

resource "aws_iot_topic_rule" "stg_power_generation_module_calling_lambda_rule" {
  for_each = var.topics

  name        = "mqtt_to_lambda_rule_${each.key}"
  description = "Invoke Lambda on MQTT messages"
  sql         = "SELECT * ,topic() as topic FROM '${each.value}'"
  sql_version = "2016-03-23"
  enabled     = true

  lambda {
    function_arn = var.stg_power_data_registration_lambda_arn
  }

}

resource "aws_lambda_permission" "stg_grant_calling_lambda_permission_to_power_generation_module" {
  for_each = var.topics

  statement_id  = "AllowExecutionFromIoT_${each.key}"
  action        = "lambda:InvokeFunction"
  function_name = var.stg_power_data_registration_lambda_function_name
  principal     = "iot.amazonaws.com"
  source_arn    = aws_iot_topic_rule.stg_power_generation_module_calling_lambda_rule[each.key].arn
}
