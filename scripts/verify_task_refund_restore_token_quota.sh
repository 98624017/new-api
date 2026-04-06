#!/usr/bin/env bash
set -euo pipefail

IMAGE="${1:-new-api:verify-20260406}"
PORT_BASE="${PORT_BASE:-38120}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MOCK_PORT_OFFSET="${MOCK_PORT_OFFSET:-1000}"

cleanup() {
  if [[ -n "${CURRENT_CID:-}" ]]; then
    docker rm -f "$CURRENT_CID" >/dev/null 2>&1 || true
    CURRENT_CID=""
  fi
  if [[ -n "${CURRENT_MOCK_PID:-}" ]]; then
    kill "${CURRENT_MOCK_PID}" >/dev/null 2>&1 || true
    wait "${CURRENT_MOCK_PID}" >/dev/null 2>&1 || true
    CURRENT_MOCK_PID=""
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

curl_json_retry() {
  local url="$1"
  shift
  local response status body
  for _ in $(seq 1 5); do
    response="$(curl -sS -w $'\n%{http_code}' "$@" "$url")" || {
      sleep 1
      continue
    }
    status="${response##*$'\n'}"
    body="${response%$'\n'*}"
    if [[ "$status" == "200" ]]; then
      printf "%s" "$body"
      return 0
    fi
    if [[ "$status" == "429" ]]; then
      sleep 1
      continue
    fi
    echo "$body" >&2
    return 1
  done
  echo "request hit repeated 429: ${url}" >&2
  return 1
}

get_wallet_quota() {
  local port="$1"
  local cookie="$2"
  curl_json_retry "http://127.0.0.1:${port}/api/user/self" -b "$cookie" -H 'New-Api-User: 1' | extract_json_int quota
}

get_token_available() {
  local port="$1"
  curl_json_retry "http://127.0.0.1:${port}/api/usage/token/" -H 'Authorization: Bearer sk-refundtasktoken' | extract_json_int total_available
}

wait_http_ready() {
  local url="$1"
  for _ in $(seq 1 30); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

run_case() {
  local restore_flag="$1"
  local port="$2"
  local mock_port="$3"
  local expected_wallet=1000
  local expected_token=300
  if [[ "$restore_flag" == "true" ]]; then
    expected_token=500
  fi

  CURRENT_DATA_DIR="$(mktemp -d /tmp/new-api-refund-${restore_flag}-XXXXXX)"
  chmod 0777 "${CURRENT_DATA_DIR}"
  touch "${CURRENT_DATA_DIR}/verify.db"
  chmod 0666 "${CURRENT_DATA_DIR}/verify.db"
  (
    cd "$ROOT_DIR"
    go run ./scripts/mock_video_failure_server.go -port "${mock_port}" >"${CURRENT_DATA_DIR}/mock-server.log" 2>&1
  ) &
  CURRENT_MOCK_PID=$!
  wait_http_ready "http://127.0.0.1:${mock_port}/healthz"

  (cd "$ROOT_DIR" && go run ./scripts/seed_task_refund_fixture.go -mode seed -db "${CURRENT_DATA_DIR}/verify.db" -base-url "http://host.docker.internal:${mock_port}") >/dev/null
  CURRENT_CID="$(docker run -d --rm \
    --user "$(id -u):$(id -g)" \
    --add-host host.docker.internal:host-gateway \
    -p "${port}:3000" \
    -e SQLITE_PATH='/data/verify.db?_busy_timeout=30000' \
    -e SESSION_SECRET=verify-refund-${restore_flag} \
    -e TASK_TIMEOUT_MINUTES=0 \
    -e TASK_REFUND_RESTORE_TOKEN_QUOTA="${restore_flag}" \
    -v "${CURRENT_DATA_DIR}:/data" \
    "${IMAGE}")"

  wait_http_ready "http://127.0.0.1:${port}/api/status"

  local cookie="${CURRENT_DATA_DIR}/user.cookie"
  curl -fsS -c "$cookie" -X POST "http://127.0.0.1:${port}/api/user/login" \
    -H "Content-Type: application/json" \
    --data '{"username":"patchuser","password":"Password123"}' >/dev/null

  local wallet_before token_before task_hits_before
  wallet_before="$(get_wallet_quota "$port" "$cookie")"
  token_before="$(get_token_available "$port")"
  task_hits_before="$(curl -fsS "http://127.0.0.1:${mock_port}/stats" | extract_json_int task_fetch_hits)"

  local wallet_after="" token_after="" task_hits_after="" inspect_out=""
  for _ in $(seq 1 45); do
    sleep 1
    task_hits_after="$(curl -fsS "http://127.0.0.1:${mock_port}/stats" | extract_json_int task_fetch_hits)"
    if [[ "${task_hits_after:-0}" -ge 1 ]]; then
      break
    fi
  done

  for _ in $(seq 1 10); do
    wallet_after="$(get_wallet_quota "$port" "$cookie")"
    token_after="$(get_token_available "$port")"
    if [[ "$wallet_after" == "$expected_wallet" && "$token_after" == "$expected_token" ]]; then
      break
    fi
    sleep 1
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
  if [[ "${task_hits_before}" != "0" ]]; then
    echo "unexpected task_hits_before: ${task_hits_before}" >&2
    docker logs "$CURRENT_CID" >&2 || true
    cat "${CURRENT_DATA_DIR}/mock-server.log" >&2 || true
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
  if [[ "${task_hits_after:-0}" -lt 1 ]]; then
    echo "video polling did not hit mock upstream, task_hits_after=${task_hits_after:-0}" >&2
    docker logs "$CURRENT_CID" >&2 || true
    cat "${CURRENT_DATA_DIR}/mock-server.log" >&2 || true
    return 1
  fi

  docker rm -f "$CURRENT_CID" >/dev/null 2>&1 || true
  CURRENT_CID=""
  if [[ -n "${CURRENT_MOCK_PID:-}" ]]; then
    kill "${CURRENT_MOCK_PID}" >/dev/null 2>&1 || true
    wait "${CURRENT_MOCK_PID}" >/dev/null 2>&1 || true
    CURRENT_MOCK_PID=""
  fi

  inspect_out="$(cd "$ROOT_DIR" && go run ./scripts/seed_task_refund_fixture.go -mode inspect -db "${CURRENT_DATA_DIR}/verify.db")"
  local task_status
  task_status="$(printf "%s\n" "$inspect_out" | awk -F= '/^TASK_STATUS=/{print $2}')"

  echo "CASE=${restore_flag}"
  echo "wallet_before=${wallet_before}"
  echo "wallet_after=${wallet_after}"
  echo "token_before=${token_before}"
  echo "token_after=${token_after}"
  echo "task_hits_after=${task_hits_after}"
  printf "%s\n" "$inspect_out"

  if [[ "$task_status" != "FAILURE" ]]; then
    echo "task status mismatch, expect FAILURE, got ${task_status}" >&2
    return 1
  fi

  rm -rf "$CURRENT_DATA_DIR" >/dev/null 2>&1 || true
  CURRENT_DATA_DIR=""
}

run_case false "${PORT_BASE}" "$((PORT_BASE + MOCK_PORT_OFFSET))"
run_case true "$((PORT_BASE + 1))" "$((PORT_BASE + MOCK_PORT_OFFSET + 1))"
