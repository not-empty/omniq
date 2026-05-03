# Scenario 18: Grouped Reap Expired

## Purpose

Validate grouped `reap_expired` follow-through.

## Validate

- retryable grouped expired jobs move to `delayed`
- terminal grouped expired jobs move to `failed`
- grouped inflight is decremented during reap
- groups are re-added to `groups:ready` when more grouped jobs remain

## Expected Outcome

- `reap_expired(...)` returns `2`
- `alpha` first job becomes `delayed`
- `beta` first job becomes `failed`
- both groups have `inflight=0` immediately after reap
- both groups are present in `groups:ready`
- the next reserves return the second waiting jobs from `alpha` and `beta`

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-18-grouped-reap-expired/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-18-grouped-reap-expired/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-18-grouped-reap-expired/go/run.go)
- [php/run.php](/home/disarli/Downloads/omniq-core/validation/scenario-18-grouped-reap-expired/php/run.php)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s18-python python /workspace/omniq/validation/scenario-18-grouped-reap-expired/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s18-node npx tsx /workspace/omniq/validation/scenario-18-grouped-reap-expired/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-18-grouped-reap-expired/go && go mod tidy && QUEUE=validation-s18-go go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && QUEUE=validation-s18-php php /workspace/omniq/validation/scenario-18-grouped-reap-expired/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `reaped_count`
- `alpha_inflight_after_reap`
- `beta_inflight_after_reap`
- `alpha_ready_after_reap`
- `beta_ready_after_reap`
- `next_job_ids`
