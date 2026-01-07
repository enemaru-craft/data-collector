# ローカル開発環境

AWS リソースを LocalStack + Docker Compose でローカル再現します。Terraform は使いません。

## 起動手順

```bash
# 1. コンテナ起動
cd local-environment
docker compose up -d

# 2. LocalStack に Lambda/DynamoDB/API Gateway を作成
chmod +x scripts/bootstrap_localstack.sh
bash scripts/bootstrap_localstack.sh
```

## コード変更時の反映

Lambda コードを変更したら、各ディレクトリで `make local` を実行すると即座に LocalStack に反映されます。

```bash
cd lambda/management-device-and-world-data-lambda
make local

cd lambda/power-data-registration-lambda
make local
```

## サービス構成

| サービス | ポート | 説明 |
|---------|--------|------|
| LocalStack | 4566 | Lambda, DynamoDB, API Gateway |
| PostgreSQL | 5432 | DB (user: postgres, pass: postgres, db: stg) |
| Mosquitto | 1883, 9001 | MQTT ブローカー (1883=MQTT, 9001=WebSocket) |
| mqtt-bridge | - | Mosquitto → Lambda を連携するブリッジ |

## MQTT → Lambda 連携

`mqtt-bridge` コンテナが `register/power` トピックを購読し、メッセージ受信時に LocalStack の `stg_power_data_registration_lambda` を invoke します。

```bash
# テスト (Mosquittoにpublish)
mosquitto_pub -h localhost -p 1883 -t register/power -m '{"device_id":"dev1","session_id":"sess1","power":100}'
```

## API Gateway エンドポイント

bootstrap 実行後に表示される URL を使用:
```
http://localhost:4566/restapis/<api_id>/$default/_user_request_/get-latest-power?device_type=solar&session_id=1
```

## 停止・リセット

```bash
# 停止
docker compose down

# データも削除してリセット
docker compose down -v
```
