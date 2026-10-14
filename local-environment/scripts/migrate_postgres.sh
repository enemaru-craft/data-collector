#!/bin/bash
set -euo pipefail

# PostgreSQL マイグレーションスクリプト (golang-migrate 使用)
# このスクリプトはローカル開発環境でマイグレーションを適用します

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MIGRATIONS_DIR="$PROJECT_ROOT/migrations"
DOCKER_COMPOSE_DIR="$PROJECT_ROOT/local-environment"

# PostgreSQL接続情報
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-stg}
DB_SSL_MODE=${DB_SSL_MODE:-disable}

DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSL_MODE}"

echo "[INFO] PostgreSQL マイグレーションを開始 (golang-migrate)..."

cd "$DOCKER_COMPOSE_DIR"

pg_ready() {
  if [ "${SETUP_CONTAINER:-0}" = "1" ]; then
    pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER"
  else
    docker compose exec -T postgres pg_isready -U "$DB_USER"
  fi
}

psql_cmd() {
  if [ "${SETUP_CONTAINER:-0}" = "1" ]; then
    PGPASSWORD="$DB_PASSWORD" PGSSLMODE="$DB_SSL_MODE" \
      psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" "$@"
  else
    docker compose exec -T postgres psql -U "$DB_USER" "$@"
  fi
}

# PostgreSQLが起動するまで待機
echo "[INFO] PostgreSQL の起動を待機中..."
for i in {1..30}; do
  if pg_ready > /dev/null 2>&1; then
    echo "[OK] PostgreSQL is ready"
    break
  fi
  if [ $i -eq 30 ]; then
    echo "[ERROR] PostgreSQL の起動がタイムアウトしました"
    exit 1
  fi
  sleep 1
done

# stg データベースが存在するか確認・作成
database_list=$(psql_cmd -lqt)
if ! printf '%s\n' "$database_list" | cut -d \| -f 1 | grep -qw "$DB_NAME"; then
  echo "[INFO] Creating database '$DB_NAME'..."
  psql_cmd -c "CREATE DATABASE \"$DB_NAME\";"
fi

# golang-migrate がインストールされているか確認
if ! command -v migrate &> /dev/null; then
  echo "[WARN] golang-migrate がインストールされていません"
  echo "[INFO] インストール方法:"
  echo "  macOS:   brew install golang-migrate"
  echo "  Linux:   curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz && sudo mv migrate /usr/local/bin/"
  echo "  go:      go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
  exit 1
fi

# マイグレーションを実行
echo "[INFO] マイグレーションを適用中..."
migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up

echo "[OK] マイグレーション完了"
echo ""

# テーブル一覧を表示
echo "[INFO] 作成されたテーブル:"
psql_cmd -d "$DB_NAME" -c "\dt" | grep -E "^\s*public\s*\|" | awk -F'|' '{print "  - " $2}' | sed 's/^ //'
