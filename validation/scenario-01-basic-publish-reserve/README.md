# Scenario 01: Basic Publish And Reserve

## Purpose

Validate the base OmniQ flow:

- publish structured payload
- job enters runnable state
- reserve returns a lease token
- job transitions to active correctly

This is the smallest full-path scenario and should be considered the baseline for all SDK parity work.

## Validate

- publish accepts structured JSON payloads
- publish rejects raw scalar payloads
- job hash is created
- job enters `{Q}:wait`
- reserve returns `JOB`
- reserve returns `lease_token`
- attempt increments correctly
- active lane and indexes are updated
- stats reflect the transition from wait to active

## Redis Checks

- `{Q}:wait`
- `{Q}:job:{job_id}`
- `{Q}:active`
- `{Q}:idx:wait`
- `{Q}:idx:active`
- `{Q}:stats`
- `omniq:queues`

## Suggested Queue

- `validation-basic-001`

## Suggested Payload

```json
{
  "kind": "basic-reserve",
  "source": "validation",
  "value": 1
}
```

## Expected Outcome

- publish returns `job_id`
- reserve returns one job with:
  - same `job_id`
  - same payload string
  - `attempt = 1`
  - non-empty `lease_token`
- job hash state becomes `active`
- wait lane no longer contains the job
- active zset contains the job with `lock_until_ms` score

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-01-basic-publish-reserve/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-01-basic-publish-reserve/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-01-basic-publish-reserve/go/run.go)

Each runner validates:

- invalid scalar publish is rejected
- structured publish succeeds
- reserve returns the same job with a lease token

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-basic-python python /workspace/omniq/validation/scenario-01-basic-publish-reserve/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-basic-node npx tsx /workspace/omniq/validation/scenario-01-basic-publish-reserve/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc 'cd /workspace/omniq/validation/scenario-01-basic-publish-reserve/go && QUEUE=validation-basic-go go run .'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `invalid_publish_rejected`
- `job_id`
- `reserve`

This makes it easier to compare the three SDKs directly.
