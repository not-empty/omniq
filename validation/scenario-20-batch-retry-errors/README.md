# Scenario 20: Batch Retry Error Matrix

## Purpose

Validate `retry_failed_batch` per-job result parity.

## Validate

- `OK` result for a failed job
- `NO_JOB` result for a missing job id
- `NOT_FAILED` result for a job that exists but is not in `failed`

## Expected Outcome

- batch result order is stable
- the failed job returns `OK` and moves back to `wait`
- the missing job returns `ERR/NO_JOB`
- the waiting job returns `ERR/NOT_FAILED`

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-20-batch-retry-errors/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-20-batch-retry-errors/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-20-batch-retry-errors/go/run.go)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s20-python python /workspace/omniq/validation/scenario-20-batch-retry-errors/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s20-node npx tsx /workspace/omniq/validation/scenario-20-batch-retry-errors/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-20-batch-retry-errors/go && go mod tidy && QUEUE=validation-s20-go go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `batch_retry_results`
- `retried_job_state`
