# Scenario 08: Pause And Resume

## Purpose

Validate queue pause semantics.

## Validate

- pause creates only the paused flag
- reserve returns `PAUSED`
- active jobs keep running
- resume removes the flag

## Redis Checks

- `{Q}:paused`
- active state before and after pause

## Expected Outcome

- one job can already be active before pause
- pause creates the paused flag
- `is_paused` returns true
- `reserve` returns `PAUSED` while the flag exists
- active job remains active during pause
- resume removes the flag
- next reserve after resume returns the waiting job

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-08-pause-resume/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-08-pause-resume/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-08-pause-resume/go/run.go)

Each runner validates:

- pause is flag-only
- reserve returns `PAUSED`
- resume re-enables reserve

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s08-python python /workspace/omniq/validation/scenario-08-pause-resume/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s08-node npx tsx /workspace/omniq/validation/scenario-08-pause-resume/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-08-pause-resume/go && go mod tidy && QUEUE=validation-s08-go go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `paused_before`
- `paused_after_pause`
- `paused_after_resume`
- `paused_reserve_status`
- `first_reserved_job_id`
- `second_reserved_job_id`
