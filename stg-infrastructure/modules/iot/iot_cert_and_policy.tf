# 証明書作成
resource "aws_iot_certificate" "stg_power_generation_module_cert" {
  active = true
}

# ポリシー設定
resource "aws_iot_policy" "stg_allow_power_generation_module_to_all_iot_actions" {
  name = "stg_allow_power_generation_module_to_all_iot_actions"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "iot:*"
        ]
        Resource = "*"
      }
    ]
  })
}

# ポリシーを証明書にアタッチ
resource "aws_iot_policy_attachment" "stg_attatch_all_actions_policy_to_power_generation_module_cert" {
  policy = aws_iot_policy.stg_allow_power_generation_module_to_all_iot_actions.name
  target = aws_iot_certificate.stg_power_generation_module_cert.arn
}

# 証明書をモノにアタッチ
resource "aws_iot_thing_principal_attachment" "stg_attach_cert_to_power_generation_module" {
  principal = aws_iot_certificate.stg_power_generation_module_cert.arn
  thing     = aws_iot_thing.stg_power_generation_module.name
}


# 一旦ローカルに証明書などを保存
resource "local_file" "stg_cert_pem" {
  content  = aws_iot_certificate.stg_power_generation_module_cert.certificate_pem
  filename = "${path.module}/stg_iot_cert.pem"
}

resource "local_file" "stg_private_key" {
  content  = aws_iot_certificate.stg_power_generation_module_cert.private_key
  filename = "${path.module}/stg_iot_private.key"
}

resource "local_file" "stg_public_key" {
  content  = aws_iot_certificate.stg_power_generation_module_cert.public_key
  filename = "${path.module}/stg_iot_public.key"
}
