#!/usr/bin/env bash
set -euo pipefail

export SETUP_CONTAINER=1 SKIP_BUILD=1 SKIP_NGINX_RELOAD=1
export LAMBDA_ARCHITECTURE="${LAMBDA_ARCHITECTURE:-$(cat /opt/lambda-architecture)}"

cd /opt/data-collector/local-environment
bash scripts/migrate_postgres.sh
bash scripts/bootstrap_localstack.sh

echo "[DONE] Setup completed. Run 'docker compose restart nginx' on the host."
