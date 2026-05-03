# Scenario 02: Heartbeat

## Purpose

Validate lease extension semantics after a successful reserve.

## Validate

- valid heartbeat extends `lock_until_ms`
- invalid token fails with contract-visible reason
- active zset score follows the lease extension

## Redis Checks

- `{Q}:job:{job_id}`
- `{Q}:active`

## Expected Outcome

- first heartbeat with current `lease_token` succeeds
- new `lock_until_ms` is greater than or equal to the previous value
- heartbeat with wrong token fails with `TOKEN_MISMATCH`

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-02-heartbeat/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-02-heartbeat/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-02-heartbeat/go/run.go)
- [php/run.php](/Users/disarli/Documents/ops/omniq/validation/scenario-02-heartbeat/php/run.php)

Each runner validates:

- reserve returns a valid lease token
- heartbeat extends the lease
- invalid token heartbeat fails with a contract-visible reason

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s02-python python /workspace/omniq/validation/scenario-02-heartbeat/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s02-node npx tsx /workspace/omniq/validation/scenario-02-heartbeat/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-02-heartbeat/go && go mod tidy && QUEUE=validation-s02-go go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && QUEUE=validation-s02-php php /workspace/omniq/validation/scenario-02-heartbeat/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `job_id`
- `initial_lock_until_ms`
- `heartbeat_lock_until_ms`
- `heartbeat_extended`
- `invalid_token_error`
- `invalid_token_contains_token_mismatch`
