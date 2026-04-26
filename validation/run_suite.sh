#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VALIDATION_DIR="$ROOT_DIR/validation"
RESULTS_ROOT="$VALIDATION_DIR/results"
RUN_TAG="$(date +%Y%m%d-%H%M%S)"
RESULTS_DIR="$RESULTS_ROOT/$RUN_TAG"

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
    12|24) echo "PREFIX" ;;
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
  local env_key env_value
  env_key="$(env_key_for "$num")"
  env_value="$(env_value_for "$num" "python")"
  docker compose exec -T omniq-python sh -lc \
    "${env_key}=${env_value} python /workspace/omniq/validation/${scenario}/python/run.py"
}

run_node() {
  local scenario="$1"
  local num="$2"
  local env_key env_value
  env_key="$(env_key_for "$num")"
  env_value="$(env_value_for "$num" "node")"
  docker compose exec -T omniq-node sh -lc \
    "${env_key}=${env_value} npx tsx /workspace/omniq/validation/${scenario}/node/run.ts"
}

run_go() {
  local scenario="$1"
  local num="$2"
  local env_key env_value
  env_key="$(env_key_for "$num")"
  env_value="$(env_value_for "$num" "go")"
  docker compose exec -T omniq-go sh -lc \
    "export PATH=/usr/bin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/${scenario}/go && /usr/bin/go mod tidy >/dev/null 2>&1 && ${env_key}=${env_value} /usr/bin/go run ."
}

run_one() {
  local sdk="$1"
  local scenario="$2"
  local num="$3"
  local outfile="$RESULTS_DIR/${num}-${sdk}.log"

  echo "Running ${scenario} on ${sdk}..."
  if "run_${sdk}" "$scenario" "$num" >"$outfile" 2>&1; then
    echo "PASS ${scenario} ${sdk}"
    return 0
  fi

  echo "FAIL ${scenario} ${sdk}"
  return 1
}

main() {
  mkdir -p "$RESULTS_DIR"

  echo "Results directory: $RESULTS_DIR"
  echo "Ensuring Docker services are up..."
  (cd "$ROOT_DIR" && docker compose up -d omniq-redis omniq-python omniq-node omniq-go >/dev/null)

  local scenarios=()
  while IFS= read -r line; do
    scenarios+=("$line")
  done < <(resolve_scenarios "$@")

  local failures=0
  local scenario num
  for scenario in "${scenarios[@]}"; do
    num="${scenario#scenario-}"
    num="${num%%-*}"

    run_one python "$scenario" "$num" || failures=$((failures + 1))
    run_one node "$scenario" "$num" || failures=$((failures + 1))
    run_one go "$scenario" "$num" || failures=$((failures + 1))
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
