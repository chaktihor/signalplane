#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export SIGNALPLANE_URL="${SIGNALPLANE_URL:-http://127.0.0.1:4318}"
export SIGNALPLANE_TOKEN="${SIGNALPLANE_TOKEN:-dev-token}"

echo "SignalPlane test target: ${SIGNALPLANE_URL}"

run_step() {
  local name="$1"
  shift
  echo
  echo "==> ${name}"
  "$@"
}

run_step "Go backend API" go run "${ROOT_DIR}/go-backend-api/main.go"
run_step "Node microservice" node "${ROOT_DIR}/node-microservice/app.mjs"
run_step "Python web backend" python3 "${ROOT_DIR}/python-web-backend/app.py"
run_step "Python worker" python3 "${ROOT_DIR}/python-worker/worker.py"
run_step "Database dependency simulator" python3 "${ROOT_DIR}/dependency-db-simulator/db_simulator.py"

run_step "C host probe build" cc -O2 -Wall -Wextra -o "${ROOT_DIR}/c-host-probe/host_probe" "${ROOT_DIR}/c-host-probe/host_probe.c"
run_step "C host probe emit" "${ROOT_DIR}/c-host-probe/host_probe"

run_step "Kubernetes workload metadata" python3 "${ROOT_DIR}/kubernetes-workload/workload.py"
run_step "Uptime target monitor" python3 "${ROOT_DIR}/uptime-target/register_monitor.py"

echo
echo "All test applications emitted telemetry."
echo "Open ${SIGNALPLANE_URL} to inspect the dashboard."

