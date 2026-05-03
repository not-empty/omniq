# Scenario 24: Queue Discovery And Sparse State

## Purpose

Validate monitor behavior when queue discovery is driven by sparse `*:stats` state.

## Validate

- `scan_queues`
- `stats(queue)`
- `stats_many([...])`

## Expected Outcome

- queues with sparse stats are still discoverable
- missing stats default cleanly to zero values
- paused status is derived correctly even when stats are missing
- partial stats do not break `stats` or `stats_many`

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](C:/Users/disarli/Documents/ops/omniq/validation/scenario-24-queue-registry-sparse/python/run.py)
- [node/run.ts](C:/Users/disarli/Documents/ops/omniq/validation/scenario-24-queue-registry-sparse/node/run.ts)
- [go/run.go](C:/Users/disarli/Documents/ops/omniq/validation/scenario-24-queue-registry-sparse/go/run.go)
- [php/run.php](/home/disarli/Downloads/omniq-core/validation/scenario-24-queue-registry-sparse/php/run.php)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'PREFIX=validation-s24-python python /workspace/omniq/validation/scenario-24-queue-registry-sparse/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'PREFIX=validation-s24-node npx tsx /workspace/omniq/validation/scenario-24-queue-registry-sparse/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/go/bin:/usr/bin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-24-queue-registry-sparse/go && /usr/local/go/bin/go mod tidy >/dev/null 2>&1 && PREFIX=validation-s24-go /usr/local/go/bin/go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && PREFIX=validation-s24-php php /workspace/omniq/validation/scenario-24-queue-registry-sparse/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queues_found`
- `stats_empty`
- `stats_partial`
- `stats_paused`
- `stats_many`
