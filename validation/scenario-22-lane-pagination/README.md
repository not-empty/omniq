# Scenario 22: Lane Pagination And Reverse Ordering

## Purpose

Validate stable pagination and reverse ordering for lane monitor views.

## Validate

- `lane_page("wait")` across multiple forward pages
- `lane_page("wait", reverse=True)` across multiple reverse pages
- `lane_page("delayed")` across multiple forward pages
- `lane_page("delayed", reverse=True)` across multiple reverse pages

## Expected Outcome

- forward pages preserve idx-score order
- reverse pages return the exact inverse order
- page boundaries are stable across SDKs
- Redis raw lane indexes match the monitor lane pages

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](C:/Users/disarli/Documents/ops/omniq/validation/scenario-22-lane-pagination/python/run.py)
- [node/run.ts](C:/Users/disarli/Documents/ops/omniq/validation/scenario-22-lane-pagination/node/run.ts)
- [go/run.go](C:/Users/disarli/Documents/ops/omniq/validation/scenario-22-lane-pagination/go/run.go)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s22-python python /workspace/omniq/validation/scenario-22-lane-pagination/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s22-node npx tsx /workspace/omniq/validation/scenario-22-lane-pagination/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/bin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-22-lane-pagination/go && QUEUE=validation-s22-go /usr/bin/go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `stats`
- `wait_forward_pages`
- `wait_reverse_pages`
- `delayed_forward_pages`
- `delayed_reverse_pages`
- `idx_wait_raw`
- `idx_delayed_raw`
