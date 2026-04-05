# Scenario 06: Promote Delayed

## Purpose

Validate delayed-job promotion back to runnable lanes.

## Validate

- due delayed jobs are promoted
- ungrouped jobs return to `{Q}:wait`
- grouped jobs return to `{Q}:g:{gid}:wait`
- indexes and counters remain aligned

## Redis Checks

- `{Q}:delayed`
- `{Q}:wait`
- `{Q}:g:{gid}:wait`
- `{Q}:idx:delayed`
- `{Q}:idx:wait`
- `{Q}:stats`

## Expected Outcome

- `publish(..., due_ms=...)` places the job in `delayed`
- `promote_delayed(...)` returns `1`
- the delayed entry is removed
- the job moves to the runnable lane
- stats move from `delayed=1` to waiting state

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-06-promote-delayed/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-06-promote-delayed/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-06-promote-delayed/go/run.go)

Each runner validates:

- delayed publish succeeds
- promote count is correct
- promotion uses an explicit deterministic due timestamp

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s06-python python /workspace/omniq/validation/scenario-06-promote-delayed/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s06-node npx tsx /workspace/omniq/validation/scenario-06-promote-delayed/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-06-promote-delayed/go && go mod tidy && QUEUE=validation-s06-go go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `job_id`
- `scheduled_due_ms`
- `promoted_count`
