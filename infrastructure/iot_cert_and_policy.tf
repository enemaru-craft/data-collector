# 証明書作成
resource "aws_iot_certificate" "cert" {
  active = true
}

# ポリシー設定
resource "aws_iot_policy" "pubsub" {
  name = "PubSubToAnyTopic"

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
resource "aws_iot_policy_attachment" "attach_policy_to_cert" {
  policy = aws_iot_policy.pubsub.name
  target = aws_iot_certificate.cert.arn
}

# 証明書をモノにアタッチ
resource "aws_iot_thing_principal_attachment" "attach_cert_to_thing" {
  principal = aws_iot_certificate.cert.arn
  thing     = aws_iot_thing.mqtt_test.name
}


# 一旦ローカルに証明書などを保存
resource "local_file" "cert_pem" {
  content  = aws_iot_certificate.cert.certificate_pem
  filename = "${path.module}/iot_cert.pem"
}

resource "local_file" "private_key" {
  content  = aws_iot_certificate.cert.private_key
  filename = "${path.module}/iot_private.key"
}

resource "local_file" "public_key" {
  content  = aws_iot_certificate.cert.public_key
  filename = "${path.module}/iot_public.key"
}
