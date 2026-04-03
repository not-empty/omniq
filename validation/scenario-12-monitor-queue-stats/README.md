# Scenario 12: Monitor Queue Stats

## Purpose

Validate queue-level monitoring views.

## Validate

- queue discovery via `omniq:queues`
- queue name normalization
- stats counters
- paused derivation
- `stats_many`

## Expected Outcome

- Python, Node, and Go should return equivalent queue and stats views for the same Redis data

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-12-monitor-queue-stats/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-12-monitor-queue-stats/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-12-monitor-queue-stats/go/run.go)

Each runner seeds two queues:

- a paused waiting queue
- a mixed queue with one active job, one delayed job, and one retained completed job

Then it validates:

- `list_queues`
- `stats(queue)`
- `stats_many([queueA, queueB])`

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'PREFIX=validation-s12-python python /workspace/omniq/validation/scenario-12-monitor-queue-stats/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'PREFIX=validation-s12-node npx tsx /workspace/omniq/validation/scenario-12-monitor-queue-stats/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-12-monitor-queue-stats/go && go mod tidy && PREFIX=validation-s12-go go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queues_found`
- `stats_a`
- `stats_b`
- `stats_many`
