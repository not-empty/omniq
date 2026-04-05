# Scenario 24: Queue Registry And Sparse State

## Purpose

Validate monitor behavior when `omniq:queues` contains queues with partial or mostly empty Redis state.

## Validate

- `list_queues`
- `stats(queue)`
- `stats_many([...])`

## Expected Outcome

- queues present only in the registry are still listed
- missing stats default cleanly to zero values
- paused status is derived correctly even when stats are missing
- partial stats do not break `stats` or `stats_many`

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](C:/Users/disarli/Documents/ops/omniq/validation/scenario-24-queue-registry-sparse/python/run.py)
- [node/run.ts](C:/Users/disarli/Documents/ops/omniq/validation/scenario-24-queue-registry-sparse/node/run.ts)
- [go/run.go](C:/Users/disarli/Documents/ops/omniq/validation/scenario-24-queue-registry-sparse/go/run.go)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'PREFIX=validation-s24-python python /workspace/omniq/validation/scenario-24-queue-registry-sparse/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'PREFIX=validation-s24-node npx tsx /workspace/omniq/validation/scenario-24-queue-registry-sparse/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/bin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-24-queue-registry-sparse/go && PREFIX=validation-s24-go /usr/bin/go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queues_found`
- `stats_empty`
- `stats_partial`
- `stats_paused`
- `stats_many`
