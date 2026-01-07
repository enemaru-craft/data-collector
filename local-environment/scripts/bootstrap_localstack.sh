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

ensure_api_gateway() {
  # LocalStack OSS では apigatewayv2 (HTTP API) は非対応なので REST API (apigateway) を使用
  local api_id
  api_id=$(awsl apigateway get-rest-apis --query "items[?name=='${API_NAME}'].id" --output text 2>/dev/null || true)
  if [ -z "$api_id" ] || [ "$api_id" = "None" ]; then
    api_id=$(awsl apigateway create-rest-api --name "$API_NAME" --query 'id' --output text)
    echo "[OK] REST API created: ${api_id}"
  else
    echo "[SKIP] REST API exists: ${api_id}"
  fi

  # Get root resource id
  local root_id
  root_id=$(awsl apigateway get-resources --rest-api-id "$api_id" --query "items[?path=='/'].id" --output text)

  # Routes (same as Terraform)
  local routes=(
    "GET:get-latest-power"
    "GET:get-latest-multiple-device-power"
    "POST:register-new-power-generation-module"
    "POST:turn-on-equipment"
    "POST:turn-off-equipment"
    "POST:get-current-world-state"
    "GET:get-power-history"
    "GET:get-game-result"
    "POST:delete-session"
  )

  for route in "${routes[@]}"; do
    local method="${route%%:*}"
    local path="${route##*:}"

    # Create resource if not exists
    local resource_id
    resource_id=$(awsl apigateway get-resources --rest-api-id "$api_id" --query "items[?pathPart=='${path}'].id" --output text 2>/dev/null || true)
    if [ -z "$resource_id" ] || [ "$resource_id" = "None" ]; then
      resource_id=$(awsl apigateway create-resource \
        --rest-api-id "$api_id" \
        --parent-id "$root_id" \
        --path-part "$path" \
        --query 'id' --output text)
      echo "[OK] Resource created: /${path}"
    fi

    # Create method if not exists
    if ! awsl apigateway get-method --rest-api-id "$api_id" --resource-id "$resource_id" --http-method "$method" >/dev/null 2>&1; then
      awsl apigateway put-method \
        --rest-api-id "$api_id" \
        --resource-id "$resource_id" \
        --http-method "$method" \
        --authorization-type NONE >/dev/null
      echo "[OK] Method added: ${method} /${path}"
    fi

    # Create integration
    awsl apigateway put-integration \
      --rest-api-id "$api_id" \
      --resource-id "$resource_id" \
      --http-method "$method" \
      --type AWS_PROXY \
      --integration-http-method POST \
      --uri "arn:aws:apigateway:${REGION}:lambda:path/2015-03-31/functions/arn:aws:lambda:${REGION}:${ACCOUNT_ID}:function:${MANAGEMENT_FN}/invocations" >/dev/null 2>&1 || true
  done

  # Deploy to 'local' stage
  awsl apigateway create-deployment --rest-api-id "$api_id" --stage-name local >/dev/null 2>&1 || true
  echo "[OK] Deployed to stage 'local'"

  # Lambda permission
  awsl lambda add-permission \
    --function-name "$MANAGEMENT_FN" \
    --action lambda:InvokeFunction \
    --statement-id apigw-invoke \
    --principal apigateway.amazonaws.com \
    --source-arn "arn:aws:execute-api:${REGION}:${ACCOUNT_ID}:${api_id}/*/*/*" >/dev/null 2>&1 || true

  echo ""
  echo "=========================================="
  echo "REST API URL: http://localhost:4566/restapis/${api_id}/local/_user_request_/"
  echo "Example: curl http://localhost:4566/restapis/${api_id}/local/_user_request_/get-latest-power?device_type=solar&session_id=1"
  echo "=========================================="
}

main() {
  wait_localstack
  build_zips
  ensure_dynamodb
  ensure_lambda "$POWER_FN" "$POWER_ZIP"
  ensure_lambda "$MANAGEMENT_FN" "$MANAGEMENT_ZIP"
  ensure_api_gateway
  echo ""
  echo "[DONE] LocalStack bootstrap completed!"
  echo ""
  echo "Usage:"
  echo "  MQTT publish to localhost:1883 topic 'register/power' -> triggers power Lambda"
  echo "  HTTP API at http://localhost:4566/restapis/<api_id>/\$default/_user_request_<path>"
}

main "$@"
