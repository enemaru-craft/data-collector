# ローカル環境のセットアップ

## 準備

- Docker Desktopを起動する。WindowsはLinuxコンテナ／WSL 2を使用し、Settings → Resources → WSL IntegrationでUbuntuとの連携を有効にする。
- リポジトリをGit cloneまたはZIPで取得する。
- 初回はインターネットに接続する。

WindowsはWSL 2のUbuntu、Macはターミナルで実行します。各コマンドが成功したら次へ進んでください。

## 初回セットアップ

リポジトリのルートで実行します。

```sh
cd local-environment
docker compose --parallel 1 up -d
docker compose run --rm --build setup
docker compose restart nginx
```

## APIの確認

オンラインのまま、テスト用のセッションID・端末IDで実行します。

```sh
curl -i -X POST http://localhost:8080/register-new-power-generation-module \
  -H 'Content-Type: application/json' \
  -d '{"sessionId":"1","deviceId":"M5-1-solar-1","deviceType":"solar"}'
curl -i -X POST http://localhost:8080/get-current-world-state \
  -H 'Content-Type: application/json' -d '{"sessionId":"1"}'
```

登録APIのHTTP 200と街の状態のJSONを確認します。オフラインで使う場合は、初回のAPI確認を済ませてからインターネットを切り、次回起動と実機のMQTT送信を試してください。実機とPCが通信するLANは必要です。

## ダミー発電データの連続送信

バックエンドを起動した後、別のターミナルでリポジトリのルートへ移動します。Python 3を用意し、初回だけ仮想環境と依存パッケージを準備します。

```sh
python3 -m venv .venv
source .venv/bin/activate
python -m pip install requests 'paho-mqtt>=2,<3'
python utils/insert_dummy_data_local.py
```

Ubuntuで仮想環境を作れない場合は、`sudo apt install python3-venv`を実行してください。

次回はリポジトリのルートで実行します。

```sh
source .venv/bin/activate
python utils/insert_dummy_data_local.py
```

停止はCtrl+Cです。実機を使うときはこの送信を停止してください。セッションIDは初期値が`1`です。変更する場合は`utils/insert_dummy_data_local.py`の`SESSION_ID`を編集し、MOD側のセッションIDも合わせます。

## 次回起動

Docker Desktopを起動し、`local-environment`で実行します。

```sh
docker compose --parallel 1 up -d
docker compose run --rm setup
docker compose restart nginx
```

起動後、実機・MODから発電データを送信してください。

## コード変更後

オンラインでsetupを再ビルドします。

```sh
docker compose run --rm --build setup
docker compose restart nginx
```

## 停止

```sh
docker compose down
```

通常の停止には`down -v`を使わないでください。PostgreSQLの保存データを削除します。

## 接続先

| 用途 | 接続先 |
| --- | --- |
| API | http://localhost:8080 |
| MQTT | localhost:1883 |
| MQTT over WebSocket | localhost:9001 |
| PostgreSQL | localhost:5432（user: postgres、pass: postgres、db: stg） |
| LocalStack | http://localhost:4566 |

実機の接続先にはPCのLAN IPを指定し、ファイアウォールとWi-Fiの端末間通信を確認してください。

## データ確認（起動中）

```sh
docker compose exec postgres psql -U postgres -d stg -c "SELECT * FROM power_logs ORDER BY timestamp DESC LIMIT 5;"
docker compose exec localstack awslocal dynamodb scan --table-name tmp_table
```

## エラー時

- **イメージ取得でEOF**: `docker compose --parallel 1 pull localstack postgres mosquitto nginx`を実行して再試行する。setupビルドも失敗したら再実行する。
- **LocalStackの認証エラー**: `docker compose config --images`で4.12.0を使っているか確認する。
- **APIが403／502**: `docker compose run --rm setup`が成功したら、`docker compose restart nginx`を実行する。
- **setupが失敗**: エラー出力と`docker compose logs --tail=100 localstack postgres`を確認する。
- **MQTTで履歴が増えない**: 登録APIを先に呼び、送信するsessionId・deviceIdを登録値と合わせる。
- **CORSエラー**: API接続先をnginxの`http://localhost:8080`にする。
