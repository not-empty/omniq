# Scenario 10: Retry And Remove Admin Operations

## Purpose

Validate lane-safe administrative operations.

## Validate

- `retry_failed`
- `retry_failed_batch`
- `remove_job`
- `remove_jobs_batch`

## Key Checks

- lane/state validation
- active-job protection
- index cleanup
- stats updates
- job hash deletion when removal succeeds

## Expected Outcome

- `retry_failed` succeeds only for jobs already in `failed`
- `retry_failed_batch` returns mixed per-job results for a valid failed job and a non-failed job
- `remove_job` rejects an active job with `ACTIVE`
- `remove_jobs_batch` returns mixed per-job results for one removable wait job and one wrong-lane delayed job
- successful removals delete the job hash and clean lane indexes

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-10-retry-remove-admin/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-10-retry-remove-admin/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-10-retry-remove-admin/go/run.go)

Each runner validates:

- single retry from `failed`
- batch retry with mixed `OK` and `NOT_FAILED`
- active-job protection on remove
- batch remove with mixed `OK` and `LANE_MISMATCH`
- single delayed removal

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s10-python python /workspace/omniq/validation/scenario-10-retry-remove-admin/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s10-node npx tsx /workspace/omniq/validation/scenario-10-retry-remove-admin/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-10-retry-remove-admin/go && go mod tidy && QUEUE=validation-s10-go go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `single_retry_state`
- `single_retry_attempt`
- `batch_retry_results`
- `remove_active_error`
- `batch_remove_results`
- `delayed_remove_result`
