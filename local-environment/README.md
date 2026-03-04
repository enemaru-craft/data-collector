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

## サービス構成

| サービス | ホスト:ポート | 説明 |
|---------|--------------|------|
| nginx | localhost:8080 | リバースプロキシ（Lambda Function URL への固定エントリポイント） |
| LocalStack | localhost:4566 | Lambda, DynamoDB |
| PostgreSQL | localhost:5432 | DB (user: postgres, pass: postgres, db: stg) |
| Mosquitto (MQTT) | localhost:1883 | MQTT ブローカー |
| Mosquitto (WebSocket) | localhost:9001 | MQTT over WebSocket |

## Lambda コード変更時の反映

```bash
# management Lambda
cd lambda/management-device-and-world-data-lambda
make local

# power Lambda
cd lambda/power-data-registration-lambda
make local
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

**解決**: nginx 経由で `http://localhost:8080/...` にアクセスしてください。直接アクセスする場合は `http://xxxxx.lambda-url.ap-northeast-1.localhost.localstack.cloud:4566/...` の形式で URL を使用してください。

### ブラウザから API にアクセスすると CORS エラー

**原因**: Lambda Function URL に直接アクセスしている

**解決**: nginx 経由（`http://localhost:8080`）でアクセスしてください。nginx が CORS ヘッダーを自動付与します。

### nginx が 502 Bad Gateway を返す

**原因**: LocalStack がまだ起動完了していない、または `bootstrap_localstack.sh` 未実行

**解決**: `docker compose logs localstack` で起動状態を確認し、`./scripts/bootstrap_localstack.sh` を実行してから nginx を再起動してください（`docker compose restart nginx`）。

### マイグレーションエラー

**原因**: golang-migrate 未インストール

**解決**: `brew install golang-migrate` を実行
