# Scenario 09: Grouped Jobs

## Purpose

Validate grouped execution behavior and per-group concurrency.

## Validate

- grouped jobs route to group wait lanes
- group limit initialization is first-writer-only
- FIFO within group is preserved
- inflight count respects the limit
- grouped and ungrouped lanes remain fair enough under reserve flow

## Redis Checks

- `{Q}:g:{gid}:wait`
- `{Q}:g:{gid}:inflight`
- `{Q}:g:{gid}:limit`
- `{Q}:groups:ready`
- `{Q}:stats`

## Expected Outcome

- the first reserve dispatches the first ready group in score order
- the second reserve dispatches the ungrouped job because lane round-robin flips
- the third reserve dispatches a different ready group while the first group is still at limit
- the fourth reserve is empty because the only remaining grouped job is blocked by inflight limit
- after `ack_success` on the first grouped job, the next reserve dispatches the second job from that same group
- the first `group_limit` value wins and later publishes do not overwrite it

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-09-grouped-jobs/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-09-grouped-jobs/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-09-grouped-jobs/go/run.go)
- [php/run.php](/Users/disarli/Documents/ops/omniq/validation/scenario-09-grouped-jobs/php/run.php)

Each runner validates:

- FIFO within group `alpha`
- independent dispatch for group `beta`
- per-group concurrency limit enforcement for `alpha`
- mixed grouped and ungrouped reserve flow

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s09-python python /workspace/omniq/validation/scenario-09-grouped-jobs/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s09-node npx tsx /workspace/omniq/validation/scenario-09-grouped-jobs/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-09-grouped-jobs/go && go mod tidy && QUEUE=validation-s09-go go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && QUEUE=validation-s09-php php /workspace/omniq/validation/scenario-09-grouped-jobs/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `group_limit_alpha`
- `reserve_order`
- `fourth_reserve_status`
- `fifth_reserve_job_id`
- `fifth_reserve_gid`
