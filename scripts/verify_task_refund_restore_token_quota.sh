#!/usr/bin/env bash
set -euo pipefail

IMAGE="${1:-new-api:verify-20260406}"
PORT_BASE="${PORT_BASE:-38120}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cleanup() {
  if [[ -n "${CURRENT_CID:-}" ]]; then
    docker rm -f "$CURRENT_CID" >/dev/null 2>&1 || true
    CURRENT_CID=""
  fi
  if [[ -n "${CURRENT_DATA_DIR:-}" && -d "${CURRENT_DATA_DIR:-}" ]]; then
    chmod -R u+w "${CURRENT_DATA_DIR}" >/dev/null 2>&1 || true
    rm -rf "$CURRENT_DATA_DIR" >/dev/null 2>&1 || true
    CURRENT_DATA_DIR=""
  fi
}

trap cleanup EXIT

extract_json_int() {
  local pattern="$1"
  grep -o "\"${pattern}\":[0-9-]*" | head -1 | cut -d: -f2
}

wait_http_ready() {
  local port="$1"
  for _ in $(seq 1 30); do
    if curl -fsS "http://127.0.0.1:${port}/api/status" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

run_case() {
  local restore_flag="$1"
  local port="$2"
  local expected_wallet=1000
  local expected_token=300
  if [[ "$restore_flag" == "true" ]]; then
    expected_token=500
  fi

  CURRENT_DATA_DIR="$(mktemp -d /tmp/new-api-refund-${restore_flag}-XXXXXX)"
  chmod 0777 "${CURRENT_DATA_DIR}"
  touch "${CURRENT_DATA_DIR}/verify.db"
  chmod 0666 "${CURRENT_DATA_DIR}/verify.db"
  (cd "$ROOT_DIR" && go run ./scripts/seed_task_refund_fixture.go -mode seed -db "${CURRENT_DATA_DIR}/verify.db") >/dev/null
  CURRENT_CID="$(docker run -d --rm \
    --user "$(id -u):$(id -g)" \
    -p "${port}:3000" \
    -e SQLITE_PATH=/data/verify.db \
    -e SESSION_SECRET=verify-refund-${restore_flag} \
    -e TASK_TIMEOUT_MINUTES=1 \
    -e TASK_REFUND_RESTORE_TOKEN_QUOTA="${restore_flag}" \
    -v "${CURRENT_DATA_DIR}:/data" \
    "${IMAGE}")"

  wait_http_ready "$port"
  sleep 1

  local cookie="${CURRENT_DATA_DIR}/user.cookie"
  curl -fsS -c "$cookie" -X POST "http://127.0.0.1:${port}/api/user/login" \
    -H "Content-Type: application/json" \
    --data '{"username":"patchuser","password":"Password123"}' >/dev/null

  local wallet_before token_before
  wallet_before="$(curl -fsS -b "$cookie" -H 'New-Api-User: 1' "http://127.0.0.1:${port}/api/user/self" | extract_json_int quota)"
  token_before="$(curl -fsS -H 'Authorization: Bearer sk-refundtasktoken' "http://127.0.0.1:${port}/api/usage/token/" | extract_json_int total_available)"

  local wallet_after="" token_after="" inspect_out=""
  for _ in $(seq 1 30); do
    sleep 1
    wallet_after="$(curl -fsS -b "$cookie" -H 'New-Api-User: 1' "http://127.0.0.1:${port}/api/user/self" | extract_json_int quota)"
    token_after="$(curl -fsS -H 'Authorization: Bearer sk-refundtasktoken' "http://127.0.0.1:${port}/api/usage/token/" | extract_json_int total_available)"
    if [[ "$wallet_after" == "$expected_wallet" && "$token_after" == "$expected_token" ]]; then
      break
    fi
  done

  if [[ "$wallet_before" != "800" ]]; then
    echo "unexpected wallet_before: ${wallet_before}" >&2
    docker logs "$CURRENT_CID" >&2 || true
    return 1
  fi
  if [[ "$token_before" != "300" ]]; then
    echo "unexpected token_before: ${token_before}" >&2
    docker logs "$CURRENT_CID" >&2 || true
    return 1
  fi
  if [[ "$wallet_after" != "$expected_wallet" ]]; then
    echo "wallet_after mismatch, expect ${expected_wallet}, got ${wallet_after}" >&2
    docker logs "$CURRENT_CID" >&2 || true
    return 1
  fi
  if [[ "$token_after" != "$expected_token" ]]; then
    echo "token_after mismatch, expect ${expected_token}, got ${token_after}" >&2
    docker logs "$CURRENT_CID" >&2 || true
    return 1
  fi

  docker rm -f "$CURRENT_CID" >/dev/null 2>&1 || true
  CURRENT_CID=""

  inspect_out="$(cd "$ROOT_DIR" && go run ./scripts/seed_task_refund_fixture.go -mode inspect -db "${CURRENT_DATA_DIR}/verify.db")"
  local task_status
  task_status="$(printf "%s\n" "$inspect_out" | awk -F= '/^TASK_STATUS=/{print $2}')"

  echo "CASE=${restore_flag}"
  echo "wallet_before=${wallet_before}"
  echo "wallet_after=${wallet_after}"
  echo "token_before=${token_before}"
  echo "token_after=${token_after}"
  printf "%s\n" "$inspect_out"

  if [[ "$task_status" != "FAILURE" ]]; then
    echo "task status mismatch, expect FAILURE, got ${task_status}" >&2
    return 1
  fi

  rm -rf "$CURRENT_DATA_DIR" >/dev/null 2>&1 || true
  CURRENT_DATA_DIR=""
}

run_case false "${PORT_BASE}"
run_case true "$((PORT_BASE + 1))"
