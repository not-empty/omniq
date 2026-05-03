#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VALIDATION_DIR="$ROOT_DIR/validation"
RESULTS_ROOT="$VALIDATION_DIR/results"
RUN_TAG="$(date +%Y%m%d-%H%M%S)"
RESULTS_DIR="$RESULTS_ROOT/$RUN_TAG"
VALIDATION_BACKENDS="${VALIDATION_BACKENDS:-standalone cluster}"

ALL_SCENARIOS=(
  "scenario-01-basic-publish-reserve"
  "scenario-02-heartbeat"
  "scenario-03-ack-success"
  "scenario-04-ack-fail-retry"
  "scenario-05-ack-fail-terminal"
  "scenario-06-promote-delayed"
  "scenario-07-reap-expired"
  "scenario-08-pause-resume"
  "scenario-09-grouped-jobs"
  "scenario-10-retry-remove-admin"
  "scenario-11-child-workflow"
  "scenario-12-monitor-queue-stats"
  "scenario-13-monitor-groups"
  "scenario-14-monitor-lanes"
  "scenario-15-error-surface"
  "scenario-16-grouped-ack-success"
  "scenario-17-grouped-ack-fail"
  "scenario-18-grouped-reap-expired"
  "scenario-19-grouped-promote-delayed"
  "scenario-20-batch-retry-errors"
  "scenario-21-batch-remove-errors"
  "scenario-22-lane-pagination"
  "scenario-23-group-pagination"
  "scenario-24-queue-registry-sparse"
  "scenario-25-noscript-recovery"
  "scenario-26-consume-drain-true"
  "scenario-27-consume-drain-false"
  "scenario-28-consume-max-attempts"
  "scenario-29-queue-name-validation"
  "scenario-30-scan-queues-discovery-rules"
  "scenario-31-multi-queue-noscript-recovery"
  "scenario-32-transport-backend-smoke"
)

pad_num() {
  printf "%02d" "$((10#$1))"
}

resolve_scenarios() {
  if [[ $# -eq 0 ]]; then
    printf "%s\n" "${ALL_SCENARIOS[@]}"
    return
  fi

  local requested=()
  local arg num padded match
  for arg in "$@"; do
    num="$(pad_num "$arg")"
    match=""
    for scenario in "${ALL_SCENARIOS[@]}"; do
      if [[ "$scenario" == "scenario-${num}-"* ]]; then
        match="$scenario"
        break
      fi
    done
    if [[ -z "$match" ]]; then
      echo "Unknown scenario number: $arg" >&2
      exit 1
    fi
    requested+=("$match")
  done

  printf "%s\n" "${requested[@]}"
}

env_key_for() {
  local num="$1"
  case "$num" in
    12|24|30) echo "PREFIX" ;;
    *) echo "QUEUE" ;;
  esac
}

env_value_for() {
  local num="$1"
  local sdk="$2"
  echo "validation-s${num}-${sdk}-auto-${RUN_TAG}"
}

run_python() {
  local scenario="$1"
  local num="$2"
  local redis_host="$3"
  local redis_mode="$4"
  local env_key env_value
  env_key="$(env_key_for "$num")"
  env_value="$(env_value_for "$num" "python")"
  docker compose exec -T omniq-python sh -lc \
    "REDIS_HOST=${redis_host} REDIS_MODE=${redis_mode} ${env_key}=${env_value} python /workspace/omniq/validation/${scenario}/python/run.py"
}

run_node() {
  local scenario="$1"
  local num="$2"
  local redis_host="$3"
  local redis_mode="$4"
  local env_key env_value
  env_key="$(env_key_for "$num")"
  env_value="$(env_value_for "$num" "node")"
  docker compose exec -T omniq-node sh -lc \
    "REDIS_HOST=${redis_host} REDIS_MODE=${redis_mode} ${env_key}=${env_value} npx tsx /workspace/omniq/validation/${scenario}/node/run.ts"
}

run_go() {
  local scenario="$1"
  local num="$2"
  local redis_host="$3"
  local redis_mode="$4"
  local env_key env_value
  env_key="$(env_key_for "$num")"
  env_value="$(env_value_for "$num" "go")"
  docker compose exec -T omniq-go sh -lc \
    "export PATH=/usr/local/go/bin:/usr/bin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/${scenario}/go && /usr/local/go/bin/go mod tidy >/dev/null 2>&1 && REDIS_HOST=${redis_host} REDIS_MODE=${redis_mode} ${env_key}=${env_value} /usr/local/go/bin/go run ."
}

run_php() {
  local scenario="$1"
  local num="$2"
  local redis_host="$3"
  local redis_mode="$4"
  local env_key env_value
  env_key="$(env_key_for "$num")"
  env_value="$(env_value_for "$num" "php")"
  docker compose exec -T omniq-php sh -lc \
    "cd /workspace/omniq-php && REDIS_HOST=${redis_host} REDIS_MODE=${redis_mode} ${env_key}=${env_value} php /workspace/omniq/validation/${scenario}/php/run.php"
}

backend_host() {
  local backend="$1"
  case "$backend" in
    standalone) echo "omniq-redis" ;;
    cluster) echo "omniq-redis-c1" ;;
    *) echo "Unknown backend: $backend" >&2; exit 1 ;;
  esac
}

wait_for_redis_resolution() {
  local backend="$1"
  local redis_host="$2"

  echo "Waiting for Redis DNS/connectivity inside SDK containers for backend=${backend} host=${redis_host}..."

  exec_probe() {
    local service="$1"
    local cmd="$2"
    local deadline=$((SECONDS + 180))

    while true; do
      if docker compose exec -T "$service" sh -lc "$cmd" >/dev/null 2>&1; then
        return 0
      fi

      if (( SECONDS >= deadline )); then
        echo "Timed out waiting for service=${service} backend=${backend}" >&2
        docker compose exec -T "$service" sh -lc "$cmd"
        return 1
      fi

      sleep 1
    done
  }

  exec_probe omniq-python "python - <<'PY'
import socket
import time

deadline = time.time() + 30
while time.time() < deadline:
    try:
        sock = socket.create_connection((\"${redis_host}\", 6379), timeout=1.0)
        sock.close()
        raise SystemExit(0)
    except OSError:
        time.sleep(0.2)
raise SystemExit(1)
PY"

  exec_probe omniq-node "node -e 'const net=require(\"net\"); const deadline=Date.now()+30000; (function probe(){ const sock=net.createConnection({host:\"${redis_host}\", port:6379}); sock.on(\"connect\", ()=>{ sock.end(); process.exit(0); }); sock.on(\"error\", ()=>{ sock.destroy(); if (Date.now() >= deadline) process.exit(1); setTimeout(probe, 200); }); })();'"

  exec_probe omniq-go "export PATH=/usr/local/go/bin:/usr/bin:/bin; cat <<'EOF' >/tmp/omniq_wait.go
package main
import (
  \"net\"
  \"os\"
  \"time\"
)
func main() {
  deadline := time.Now().Add(30 * time.Second)
  for time.Now().Before(deadline) {
    conn, err := net.DialTimeout(\"tcp\", \"${redis_host}:6379\", time.Second)
    if err == nil {
      conn.Close()
      os.Exit(0)
    }
    time.Sleep(200 * time.Millisecond)
  }
  os.Exit(1)
}
EOF
/usr/local/go/bin/go run /tmp/omniq_wait.go >/dev/null 2>&1"

  exec_probe omniq-php "php -r '\$deadline=microtime(true)+30; while (microtime(true) < \$deadline) { \$sock=@fsockopen(\"${redis_host}\", 6379, \$errno, \$errstr, 1.0); if (\$sock !== false) { fclose(\$sock); exit(0); } usleep(200000); } exit(1);'"
}

run_one() {
  local sdk="$1"
  local scenario="$2"
  local num="$3"
  local redis_host="$4"
  local redis_mode="$5"
  local outfile="$RESULTS_DIR/${redis_mode}-${num}-${sdk}.log"

  echo "Running ${scenario} on ${sdk} backend=${redis_mode}..."
  if "run_${sdk}" "$scenario" "$num" "$redis_host" "$redis_mode" >"$outfile" 2>&1; then
    echo "PASS ${scenario} ${sdk} backend=${redis_mode}"
    return 0
  fi

  echo "FAIL ${scenario} ${sdk} backend=${redis_mode}"
  return 1
}

main() {
  mkdir -p "$RESULTS_DIR"

  echo "Results directory: $RESULTS_DIR"
  echo "Ensuring Docker services are up..."
  (cd "$ROOT_DIR" && docker compose up -d omniq-redis omniq-redis-c1 omniq-redis-c2 omniq-redis-c3 omniq-redis-cluster-init omniq-python omniq-node omniq-go omniq-php >/dev/null)

  local scenarios=()
  while IFS= read -r line; do
    scenarios+=("$line")
  done < <(resolve_scenarios "$@")

  local failures=0
  local scenario num backend redis_host
  for backend in $VALIDATION_BACKENDS; do
    redis_host="$(backend_host "$backend")"
    (cd "$ROOT_DIR" && wait_for_redis_resolution "$backend" "$redis_host")

    for scenario in "${scenarios[@]}"; do
      num="${scenario#scenario-}"
      num="${num%%-*}"

      run_one python "$scenario" "$num" "$redis_host" "$backend" || failures=$((failures + 1))
      run_one node "$scenario" "$num" "$redis_host" "$backend" || failures=$((failures + 1))
      run_one go "$scenario" "$num" "$redis_host" "$backend" || failures=$((failures + 1))
      run_one php "$scenario" "$num" "$redis_host" "$backend" || failures=$((failures + 1))
    done
  done

  echo
  echo "Run finished. Logs are in: $RESULTS_DIR"
  if [[ "$failures" -gt 0 ]]; then
    echo "Failures: $failures"
    exit 1
  fi

  echo "All requested scenario runners completed successfully."
}

main "$@"
