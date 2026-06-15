#!/bin/sh
set -eu

LOG_DIR="${SIGNALPLANE_APP_LOG_DIR:-/var/log/signalplane-apps}"
LOG_FILE="${LOG_DIR}/${SIGNALPLANE_LOG_FILE:-checkout-api.log}"
SERVICE="${SIGNALPLANE_SERVICE_NAME:-agent-checkout-api}"
HOST="${SIGNALPLANE_HOST_NAME:-demo-node-1}"
ENVIRONMENT="${SIGNALPLANE_ENVIRONMENT:-production}"
REGION="${SIGNALPLANE_REGION:-local}"
VERSION="${SIGNALPLANE_SERVICE_VERSION:-agent-demo-0.1.0}"
INTERVAL_SECONDS="${SIGNALPLANE_LOG_INTERVAL_SECONDS:-2}"
FAIL_EVERY="${SIGNALPLANE_LOG_FAIL_EVERY:-5}"

mkdir -p "$LOG_DIR"

i=0
while true; do
  i=$((i + 1))
  timestamp="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  order_id="$(printf 'ord-agent-%06d' "$i")"
  trace_id="$(printf '%032x' "$i")"
  span_id="$(printf '%016x' "$i")"
  severity="info"
  message="checkout completed through local log agent"
  error_kind=""

  if [ "$FAIL_EVERY" -gt 0 ] && [ $((i % FAIL_EVERY)) -eq 0 ]; then
    severity="error"
    message="checkout failed payment gateway timeout through local log agent"
    error_kind="payment_gateway_timeout"
  fi

  printf '{"timestamp":"%s","severity":"%s","message":"%s","traceId":"%s","spanId":"%s","service":"%s","host":"%s","environment":"%s","region":"%s","version":"%s","orderId":"%s","source":"node-agent-demo","route":"/checkout","error.kind":"%s"}\n' \
    "$timestamp" "$severity" "$message" "$trace_id" "$span_id" "$SERVICE" "$HOST" "$ENVIRONMENT" "$REGION" "$VERSION" "$order_id" "$error_kind" >> "$LOG_FILE"

  sleep "$INTERVAL_SECONDS"
done
