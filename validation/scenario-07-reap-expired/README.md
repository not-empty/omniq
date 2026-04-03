# Scenario 07: Reap Expired

## Purpose

Validate stalled-job recovery from the active lane.

## Validate

- expired active jobs are reaped
- retryable jobs move to delayed
- terminal jobs move to failed
- orphaned active entries are cleaned up

## Redis Checks

- `{Q}:active`
- `{Q}:delayed`
- `{Q}:failed`
- indexes and stats

## Expected Outcome

- retryable expired job moves from `active` to `delayed`
- exhausted expired job moves from `active` to `failed`
- `reap_expired(...)` returns the number of reaped jobs
- stats reflect `active=0` plus one delayed and one failed job

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-07-reap-expired/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-07-reap-expired/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-07-reap-expired/go/run.go)

Each runner validates:

- a retryable expired active job is reaped to `delayed`
- a terminal expired active job is reaped to `failed`
- the reap count is correct

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s07-python python /workspace/omniq/validation/scenario-07-reap-expired/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s07-node npx tsx /workspace/omniq/validation/scenario-07-reap-expired/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-07-reap-expired/go && go mod tidy && QUEUE=validation-s07-go go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `reaped_count`
- `retryable_job_id`
- `terminal_job_id`
