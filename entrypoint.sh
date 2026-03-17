#!/usr/bin/env bash
set -e

export APP_NAME="autenticacion-bff"
export APP_ENV="dev"
export APP_PORT="8000"
export DEBUG="true"
export API_V1_PREFIX="/api/v1"

export KEYCLOAK_BASE_URL="http://localhost:8080"
export KEYCLOAK_REALM="demo"
export KEYCLOAK_CLIENT_ID="autenticacionWso2"
export KEYCLOAK_CLIENT_SECRET="03TvNSVWDRK0ZedfiLehJC9ZwX4ERy9f"
export KEYCLOAK_REDIRECT_URI="http://localhost:8000/api/v1/auth/callback"

export AWS_REGION="us-east-1"
export DYNAMODB_TABLE_SESIONES="sesiones"
export AWS_ACCESS_KEY_ID="dummy"
export AWS_SECRET_ACCESS_KEY="dummy"
export DYNAMODB_ENDPOINT_URL="http://localhost:8001"

gunicorn api:app \
  -k uvicorn_worker.UvicornWorker \
  -w 1 \
  -b 0.0.0.0:8000 \
  --timeout 120 \
  --access-logfile - \
  --error-logfile -
