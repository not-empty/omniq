# Scenario 14: Monitor Lane Views

## Purpose

Validate lane browsing and job inspection views.

## Validate

- `lane_page`
- `find_jobs`
- `get_job`
- `overview`

## Special Focus

- skipped missing job hashes
- score ordering
- reverse ordering
- field parity across SDKs

## Expected Outcome

- `lane_page("wait")` skips the dangling index entry whose job hash was deleted
- reverse ordering on `lane_page("wait")` matches the remaining idx score order
- `find_jobs("wait", [...])` returns only the job that still has a hash
- `get_job(existing)` returns the full job info
- `get_job(missing-hash)` returns null
- `overview()` matches the same active, delayed, failed, and completed job ids as direct lane reads

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-14-monitor-lanes/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-14-monitor-lanes/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-14-monitor-lanes/go/run.go)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s14-python python /workspace/omniq/validation/scenario-14-monitor-lanes/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s14-node npx tsx /workspace/omniq/validation/scenario-14-monitor-lanes/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-14-monitor-lanes/go && go mod tidy && QUEUE=validation-s14-go go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `wait_page`
- `wait_page_reverse`
- `find_wait`
- `get_existing`
- `get_missing`
- `overview`
