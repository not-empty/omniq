# Scenario 04: Ack Fail With Retry

## Purpose

Validate retryable failure behavior for active jobs.

## Validate

- `ack_fail` requires valid `lease_token`
- retryable job moves to delayed
- due time matches `backoff_ms`
- last error fields are stored

## Redis Checks

- `{Q}:job:{job_id}`
- `{Q}:delayed`
- `{Q}:idx:delayed`
- `{Q}:stats`

## Expected Outcome

- SDK returns retry status plus `due_ms`
- job state becomes `delayed`
- delayed zset contains the job

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-04-ack-fail-retry/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-04-ack-fail-retry/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-04-ack-fail-retry/go/run.go)

Each runner validates:

- reserve returns a lease token
- invalid token ack fails with a contract-visible reason
- valid `ack_fail` returns retry status and `due_ms`

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s04-python python /workspace/omniq/validation/scenario-04-ack-fail-retry/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s04-node npx tsx /workspace/omniq/validation/scenario-04-ack-fail-retry/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-04-ack-fail-retry/go && go mod tidy && QUEUE=validation-s04-go go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `job_id`
- `ack_fail_status`
- `due_ms`
- `invalid_token_error`
- `invalid_token_contains_token_mismatch`
