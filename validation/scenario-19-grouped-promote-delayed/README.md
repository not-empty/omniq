# Scenario 19: Grouped Promote Delayed

## Purpose

Validate grouped delayed promotion.

## Validate

- grouped delayed jobs are promoted into `g:{gid}:wait`
- `groups:ready` is populated for promoted groups
- grouped waiting counters are updated
- promoted grouped jobs become reservable in the next reserve cycle

## Expected Outcome

- `promote_delayed(...)` returns `2`
- promoted jobs leave `{Q}:delayed`
- `alpha` and `beta` are both present in `{Q}:groups:ready`
- group waiting counters reflect two promoted grouped jobs
- the next reserves return the promoted grouped jobs

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-19-grouped-promote-delayed/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-19-grouped-promote-delayed/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-19-grouped-promote-delayed/go/run.go)
- [php/run.php](/home/disarli/Downloads/omniq-core/validation/scenario-19-grouped-promote-delayed/php/run.php)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s19-python python /workspace/omniq/validation/scenario-19-grouped-promote-delayed/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s19-node npx tsx /workspace/omniq/validation/scenario-19-grouped-promote-delayed/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-19-grouped-promote-delayed/go && go mod tidy && QUEUE=validation-s19-go go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && QUEUE=validation-s19-php php /workspace/omniq/validation/scenario-19-grouped-promote-delayed/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `promoted_count`
- `alpha_ready_after_promote`
- `beta_ready_after_promote`
- `group_waiting_after_promote`
- `next_job_ids`
