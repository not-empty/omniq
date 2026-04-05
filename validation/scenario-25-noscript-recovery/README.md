# Scenario 25: Script Reload And NOSCRIPT Recovery

## Purpose

Validate that each SDK recovers cleanly from Redis script cache misses.

## Validate

- `publish` after `SCRIPT FLUSH`
- `reserve` after `SCRIPT FLUSH`
- `heartbeat` after `SCRIPT FLUSH`
- `ack_success` after `SCRIPT FLUSH`
- `promote_delayed` after `SCRIPT FLUSH`

## Expected Outcome

- each operation succeeds after the script cache is flushed
- the SDK reloads and evaluates the correct Lua script automatically
- the completed job ends in `completed`
- the promoted delayed job ends in `wait`

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](C:/Users/disarli/Documents/ops/omniq/validation/scenario-25-noscript-recovery/python/run.py)
- [node/run.ts](C:/Users/disarli/Documents/ops/omniq/validation/scenario-25-noscript-recovery/node/run.ts)
- [go/run.go](C:/Users/disarli/Documents/ops/omniq/validation/scenario-25-noscript-recovery/go/run.go)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s25-python python /workspace/omniq/validation/scenario-25-noscript-recovery/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s25-node npx tsx /workspace/omniq/validation/scenario-25-noscript-recovery/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/bin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-25-noscript-recovery/go && QUEUE=validation-s25-go /usr/bin/go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `published_job_id`
- `reserved_job_id`
- `heartbeat_lock_until_ms`
- `completed_state`
- `delayed_job_id`
- `promoted_count`
- `promoted_state`
