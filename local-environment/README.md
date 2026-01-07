# ローカル開発環境

AWS リソースを LocalStack + Docker Compose でローカル再現します。

## 前提条件

- Docker / Docker Compose
- golang-migrate (`brew install golang-migrate`)
- Go 1.21+

## クイックスタート

```bash
cd local-environment

# 1. コンテナ起動
docker compose up -d

# 2. PostgreSQL マイグレーション
./scripts/migrate_postgres.sh

# 3. Lambda / DynamoDB / Function URL を作成
./scripts/bootstrap_localstack.sh
```

bootstrap 完了後、以下のような URL が表示されます（この URL を使用）:
```
Management Lambda Function URL:
  http://xxxxx.lambda-url.ap-northeast-1.localhost.localstack.cloud:4566/
```

## サービス構成

| サービス | ホスト:ポート | 説明 |
|---------|--------------|------|
| LocalStack | localhost:4566 | Lambda, DynamoDB |
| PostgreSQL | localhost:5432 | DB (user: postgres, pass: postgres, db: stg) |
| Mosquitto (MQTT) | localhost:1883 | MQTT ブローカー |
| Mosquitto (WebSocket) | localhost:9001 | MQTT over WebSocket |

## API の使い方

### Function URL の取得

```bash
docker compose exec localstack awslocal lambda get-function-url-config \
  --function-name stg_management_device_and_world_data_lambda \
  --query 'FunctionUrl' --output text
```

### 1. デバイス登録 (必須・最初に実行)

電力データを送信する前に、デバイスを登録する必要があります。

```bash
curl -X POST 'http://<function-url>/register-new-power-generation-module' \
  -H 'Content-Type: application/json' \
  -d '{
    "sessionId": "1",
    "deviceId": "solar-001",
    "deviceType": "solar"
  }'
```

### 2. MQTT で電力データ送信

```bash
docker compose exec mosquitto mosquitto_pub \
  -h localhost -p 1883 -t register/power \
  -m '{
    "sessionId": "1",
    "deviceId": "solar-001",
    "deviceType": "solar",
    "power": 150.5,
    "gpsLat": "35.6812",
    "gpsLon": "139.7671"
  }'
```

### 3. 電力データ取得

```bash
# 単一デバイスの最新電力
curl 'http://<function-url>/get-latest-power?device_type=solar&session_id=1'

# 複数デバイスの電力合計
curl 'http://<function-url>/get-latest-multiple-device-power?device_type=solar&session_id=1'
```

### 4. 世界の状態取得

```bash
curl -X POST 'http://<function-url>/get-current-world-state' \
  -H 'Content-Type: application/json' \
  -d '{"sessionId": "1"}'
```

## Lambda コード変更時の反映

```bash
# management Lambda
cd lambda/management-device-and-world-data-lambda
make local

# power Lambda
cd lambda/power-data-registration-lambda
make local
```

## ログ確認

```bash
# mqtt-bridge (MQTT → Lambda 連携)
docker compose logs -f mqtt-bridge

# LocalStack
docker compose logs -f localstack

# PostgreSQL
docker compose logs -f postgres
```

## データ確認

```bash
# PostgreSQL のテーブル確認
docker compose exec postgres psql -U postgres -d stg -c "SELECT * FROM power_logs ORDER BY timestamp DESC LIMIT 5;"

# DynamoDB のデータ確認
docker compose exec localstack awslocal dynamodb scan --table-name tmp_table
```

## 停止・リセット

```bash
# 停止
docker compose down

# データも削除して完全リセット
docker compose down -v
```

## トラブルシューティング

### MQTT 送信しても電力データが登録されない

**原因**: デバイスが未登録

**解決**: 先に `/register-new-power-generation-module` でデバイスを登録してください。

### `localhost:4566/...` でアクセスするとエラー

**原因**: Lambda Function URL を使う必要があります

**解決**: `http://xxxxx.lambda-url.ap-northeast-1.localhost.localstack.cloud:4566/...` の形式で URL を使用してください。

### マイグレーションエラー

**原因**: golang-migrate 未インストール

**解決**: `brew install golang-migrate` を実行
