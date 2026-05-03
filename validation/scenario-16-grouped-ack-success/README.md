# Scenario 16: Grouped Ack Success

## Purpose

Validate grouped `ack_success` follow-through.

## Validate

- grouped `ack_success` decrements `{Q}:g:{gid}:inflight`
- the group is re-added to `{Q}:groups:ready` when more grouped jobs remain
- the next grouped job becomes reservable immediately

## Expected Outcome

- first reserve returns the first grouped job
- after `ack_success`, group inflight becomes `0`
- the group is present in `groups:ready`
- the second reserve returns the next job from the same group

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-16-grouped-ack-success/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-16-grouped-ack-success/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-16-grouped-ack-success/go/run.go)
- [php/run.php](/home/disarli/Downloads/omniq-core/validation/scenario-16-grouped-ack-success/php/run.php)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s16-python python /workspace/omniq/validation/scenario-16-grouped-ack-success/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s16-node npx tsx /workspace/omniq/validation/scenario-16-grouped-ack-success/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-16-grouped-ack-success/go && go mod tidy && QUEUE=validation-s16-go go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && QUEUE=validation-s16-php php /workspace/omniq/validation/scenario-16-grouped-ack-success/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `first_job_id`
- `second_job_id`
- `group_ready_after_ack`
- `group_inflight_after_ack`
