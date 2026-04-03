# Scenario 17: Grouped Ack Fail Retry And Terminal

## Purpose

Validate grouped `ack_fail` follow-through for both retryable and terminal failure.

## Validate

- grouped retryable fail decrements inflight and re-arms the group when more jobs remain
- grouped terminal fail also decrements inflight and re-arms the group when more jobs remain
- retryable grouped job moves to `delayed`
- terminal grouped job moves to `failed`

## Expected Outcome

- first grouped failure returns `RETRY`
- second grouped failure returns `FAILED`
- both groups have `inflight=0` immediately after their fail ack
- both groups are present in `groups:ready` because each still has one waiting job
- the next reserves return the second waiting jobs from `alpha` and `beta`

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-17-grouped-ack-fail/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-17-grouped-ack-fail/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-17-grouped-ack-fail/go/run.go)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s17-python python /workspace/omniq/validation/scenario-17-grouped-ack-fail/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s17-node npx tsx /workspace/omniq/validation/scenario-17-grouped-ack-fail/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-17-grouped-ack-fail/go && go mod tidy && QUEUE=validation-s17-go go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `alpha_fail_status`
- `beta_fail_status`
- `alpha_inflight_after_fail`
- `beta_inflight_after_fail`
- `alpha_ready_after_fail`
- `beta_ready_after_fail`
- `next_job_ids`
