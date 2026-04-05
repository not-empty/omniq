# Scenario 21: Batch Remove Error Matrix

## Purpose

Validate `remove_jobs_batch` per-job result parity for the full wait-lane error matrix.

## Validate

- `OK` for a removable wait job
- `NO_JOB` for a missing job id
- `NOT_IN_LANE` for a grouped wait job passed to the plain `wait` lane
- `ACTIVE` for a leased job
- `LANE_MISMATCH` for a job that exists in a different state

## Expected Outcome

- batch result order is stable
- only the plain wait job is actually removed
- the grouped wait, active, and delayed jobs remain present
- only the successful removal changes wait counters and wait indexes

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](C:/Users/disarli/Documents/ops/omniq/validation/scenario-21-batch-remove-errors/python/run.py)
- [node/run.ts](C:/Users/disarli/Documents/ops/omniq/validation/scenario-21-batch-remove-errors/node/run.ts)
- [go/run.go](C:/Users/disarli/Documents/ops/omniq/validation/scenario-21-batch-remove-errors/go/run.go)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s21-python python /workspace/omniq/validation/scenario-21-batch-remove-errors/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s21-node npx tsx /workspace/omniq/validation/scenario-21-batch-remove-errors/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/bin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-21-batch-remove-errors/go && QUEUE=validation-s21-go /usr/bin/go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `batch_remove_results`
- `job_hash_exists`
- `stats`
- `wait_len`
- `idx_wait`
- `group_wait_len`
- `groups_ready`
