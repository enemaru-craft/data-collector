#!/usr/bin/env bash
set -euo pipefail

# ========================================
# LocalStack Bootstrap Script (No Terraform)
# - Build Lambda ZIPs (same as prod)
# - Create/update Lambdas in LocalStack
# - Create DynamoDB table
# - Create HTTP API routes -> management Lambda
# ========================================

REGION=${REGION:-ap-northeast-1}
ACCOUNT_ID=000000000000
LAMBDA_ROLE_ARN="arn:aws:iam::${ACCOUNT_ID}:role/lambda-exec"
LOCALSTACK_ENDPOINT=${LOCALSTACK_ENDPOINT:-http://localhost:4566}
REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)

MANAGEMENT_DIR="$REPO_ROOT/lambda/management-device-and-world-data-lambda"
POWER_DIR="$REPO_ROOT/lambda/power-data-registration-lambda"
MANAGEMENT_ZIP="$MANAGEMENT_DIR/lambda_function.zip"
POWER_ZIP="$POWER_DIR/lambda_function.zip"

MANAGEMENT_FN=stg_management_device_and_world_data_lambda
POWER_FN=stg_power_data_registration_lambda
API_NAME=stg_device_and_world_management_api

# awslocal wrapper
awsl() {
  aws --endpoint-url "$LOCALSTACK_ENDPOINT" --region "$REGION" "$@"
}

wait_localstack() {
  echo "[INFO] Waiting for LocalStack to be ready..."
  for i in {1..30}; do
    if awsl sts get-caller-identity >/dev/null 2>&1; then
      echo "[OK] LocalStack is ready"
      return 0
    fi
    sleep 2
  done
  echo "[ERROR] LocalStack not ready after 60s" >&2
  exit 1
}

build_zips() {
  echo "[INFO] Building Lambda ZIPs..."
  (cd "$MANAGEMENT_DIR" && make zip)
  (cd "$POWER_DIR" && make zip)
  echo "[OK] ZIPs built"
}

ensure_dynamodb() {
  local table=tmp_table
  if awsl dynamodb describe-table --table-name "$table" >/dev/null 2>&1; then
    echo "[SKIP] DynamoDB table ${table} exists"
    return
  fi
  awsl dynamodb create-table \
    --table-name "$table" \
    --attribute-definitions \
      AttributeName=session_id,AttributeType=S \
      AttributeName=device_id,AttributeType=S \
    --key-schema \
      AttributeName=session_id,KeyType=HASH \
      AttributeName=device_id,KeyType=RANGE \
    --billing-mode PAY_PER_REQUEST >/dev/null
  echo "[OK] DynamoDB table ${table} created"
}

ensure_lambda() {
  local name=$1 zip=$2
  if ! awsl lambda get-function --function-name "$name" >/dev/null 2>&1; then
    awsl lambda create-function \
      --function-name "$name" \
      --runtime provided.al2023 \
      --handler main \
      --role "$LAMBDA_ROLE_ARN" \
      --timeout 30 \
      --zip-file "fileb://${zip}" >/dev/null
    echo "[OK] Lambda ${name} created"
  else
    awsl lambda update-function-code \
      --function-name "$name" \
      --zip-file "fileb://${zip}" >/dev/null
    echo "[OK] Lambda ${name} code updated"
  fi
  # 環境変数設定 (DB_HOSTはdocker-composeのservice名, sslmode=disableはローカル用)
  awsl lambda update-function-configuration \
    --function-name "$name" \
    --timeout 30 \
    --environment "Variables={DB_HOST=host.docker.internal,DB_PASSWORD=postgres,AWS_REGION=${REGION},AWS_ENDPOINT_URL=http://host.docker.internal:4566}" >/dev/null 2>&1 || true
  echo "[OK] Lambda ${name} env configured"
}

ensure_lambda_function_url() {
  # Lambda Function URL を作成 (HTTP API v2 形式のイベントを送るので Lambda コードと互換性あり)
  local name=$1

  # Function URL が存在するか確認
  local url
  url=$(awsl lambda get-function-url-config --function-name "$name" --query 'FunctionUrl' --output text 2>/dev/null || true)

  if [ -z "$url" ] || [ "$url" = "None" ]; then
    awsl lambda create-function-url-config \
      --function-name "$name" \
      --auth-type NONE \
      --cors AllowOrigins='*',AllowMethods='*',AllowHeaders='*' >/dev/null

    # パブリックアクセス許可
    awsl lambda add-permission \
      --function-name "$name" \
      --action lambda:InvokeFunctionUrl \
      --statement-id FunctionURLAllowPublicAccess \
      --principal '*' \
      --function-url-auth-type NONE >/dev/null 2>&1 || true

    url=$(awsl lambda get-function-url-config --function-name "$name" --query 'FunctionUrl' --output text)
    echo "[OK] Function URL created for ${name}: ${url}"
  else
    echo "[SKIP] Function URL exists for ${name}: ${url}"
  fi
}

main() {
  wait_localstack
  build_zips
  ensure_dynamodb
  ensure_lambda "$POWER_FN" "$POWER_ZIP"
  ensure_lambda "$MANAGEMENT_FN" "$MANAGEMENT_ZIP"
  ensure_lambda_function_url "$MANAGEMENT_FN"

  local mgmt_url
  mgmt_url=$(awsl lambda get-function-url-config --function-name "$MANAGEMENT_FN" --query 'FunctionUrl' --output text 2>/dev/null || echo "")

  echo ""
  echo "[DONE] LocalStack bootstrap completed!"
  echo ""
  echo "=========================================="
  echo "Management Lambda Function URL:"
  echo "  ${mgmt_url}"
  echo ""
  echo "Example requests:"
  echo "  curl -X POST '${mgmt_url}register-new-power-generation-module' \\"
  echo "    -H 'Content-Type: application/json' \\"
  echo "    -d '{\"sessionId\":\"1\",\"deviceId\":\"dev1\",\"deviceType\":\"solar\"}'"
  echo ""
  echo "  curl '${mgmt_url}get-latest-power?device_type=solar&session_id=1'"
  echo ""
  echo "MQTT:"
  echo "  Publish to localhost:1883 topic 'register/power' -> triggers power Lambda"
  echo "=========================================="
}

main "$@"
