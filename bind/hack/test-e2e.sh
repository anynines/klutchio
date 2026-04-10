#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

is_listening() {
  lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1
}

listener_pid() {
  lsof -t -nP -iTCP:"$1" -sTCP:LISTEN | head -n 1
}

maybe_stop_stale_listener() {
  local port="$1"
  local pattern1="$2"
  local pattern2="$3"
  local name="$4"
  local pid cmd

  if ! is_listening "$port"; then
    return
  fi

  pid="$(listener_pid "$port")"
  cmd="$(ps -p "$pid" -o command= 2>/dev/null || true)"
  if [[ "$cmd" == *"$pattern1"* ]] || [[ "$cmd" == *"$pattern2"* ]]; then
    echo "Stopping stale ${name} listener on port ${port} (pid ${pid})."
    kill -TERM "$pid" >/dev/null 2>&1 || true
    sleep 1
  fi
}

tail_kcp_log() {
  tail -n 200 .kcp/kcp.log || true
}

select_kcp_port() {
  local selected="$1"

  if is_listening "$selected"; then
    echo "Port ${selected} is already in use. Auto-selecting a free KCP port."
    for candidate in $(seq 16444 16600); do
      if ! is_listening "$candidate"; then
        selected="$candidate"
        break
      fi
    done
  fi

  echo "$selected"
}

wait_for_kcp_ready() {
  local ready_timeout="$1"
  local kcp_pid="$2"

  echo "Waiting for kcp to be ready (check .kcp/kcp.log)."
  for i in $(seq 1 "$ready_timeout"); do
    if KUBECONFIG=.kcp/admin.kubeconfig kubectl get --raw /readyz >/dev/null 2>&1; then
      echo
      return 0
    fi

    if ! kill -0 "$kcp_pid" >/dev/null 2>&1; then
      echo
      echo "kcp exited before becoming ready. Last 200 lines from .kcp/kcp.log:"
      tail_kcp_log
      return 1
    fi

    printf "."
    sleep 1
  done

  echo
  echo "Timed out waiting for kcp readiness after ${ready_timeout} seconds. Last 200 lines from .kcp/kcp.log:"
  tail_kcp_log
  return 1
}

mkdir -p .kcp

maybe_stop_stale_listener "$DEX_HTTP_PORT_VALUE" "/dex-v" "dex serve" "dex"
if is_listening "$DEX_HTTP_PORT_VALUE"; then
  echo "Port ${DEX_HTTP_PORT_VALUE} is already in use. Stop the conflicting process and retry."
  exit 1
fi

maybe_stop_stale_listener "$KCP_SECURE_PORT_VALUE" "/kcp-v" "kcp start" "kcp"
kcp_port="$(select_kcp_port "$KCP_SECURE_PORT_VALUE")"
if is_listening "$kcp_port"; then
  echo "Unable to find a free KCP port in range 16444-16600. Set KCP_SECURE_PORT=<free-port> and retry."
  exit 1
fi

echo "Using KCP secure port ${kcp_port}"
"$DEX_BIN" serve hack/dex-config-dev.yaml 2>&1 & DEX_PID=$!
"$KCP_BIN" start \
  --root-directory=.kcp \
  --kubeconfig-path=.kcp/admin.kubeconfig \
  --secure-port="$kcp_port" \
  --bind-address=127.0.0.1 \
  --external-hostname=127.0.0.1 \
  --shard-base-url="https://127.0.0.1:${kcp_port}" \
  --shard-external-url="https://127.0.0.1:${kcp_port}" \
  &>.kcp/kcp.log & KCP_PID=$!

cleanup() {
  kill -TERM "$DEX_PID" >/dev/null 2>&1 || true
  kill -TERM "$KCP_PID" >/dev/null 2>&1 || true
}
trap cleanup TERM INT EXIT

if ! wait_for_kcp_ready "$KCP_READY_TIMEOUT_SECONDS_VALUE" "$KCP_PID"; then
  exit 1
fi

test_command="KUBECONFIG=$PWD/.kcp/admin.kubeconfig GOOS=$GOOS_VALUE GOARCH=$GOARCH_VALUE $GO_TEST_CMD -race -count $COUNT_VALUE -p $E2E_PARALLELISM_VALUE -parallel $E2E_PARALLELISM_VALUE $WHAT_VALUE $TEST_ARGS_VALUE"
if ! eval "$test_command"; then
  echo "e2e tests failed. Last 200 lines from .kcp/kcp.log:"
  tail_kcp_log
  exit 1
fi

if [[ "$KEEP_E2E_ARTIFACTS_VALUE" != "1" ]]; then
  rm -rf .kcp
fi
