# Scenario 05: Ack Fail To Terminal Failure

## Purpose

Validate terminal failure when attempts are exhausted.

## Validate

- exhausted jobs move to `failed`
- no delayed schedule is created
- failure indexes and counters are updated

## Redis Checks

- `{Q}:failed`
- `{Q}:idx:failed`
- `{Q}:stats`

## Expected Outcome

- SDK returns failed status
- job state becomes `failed`

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-05-ack-fail-terminal/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-05-ack-fail-terminal/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-05-ack-fail-terminal/go/run.go)

Each runner validates:

- reserve returns a lease token
- invalid token ack fails with a contract-visible reason
- valid `ack_fail` on an exhausted job returns terminal failure

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s05-python python /workspace/omniq/validation/scenario-05-ack-fail-terminal/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s05-node npx tsx /workspace/omniq/validation/scenario-05-ack-fail-terminal/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-05-ack-fail-terminal/go && go mod tidy && QUEUE=validation-s05-go go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `job_id`
- `ack_fail_status`
- `due_ms`
- `invalid_token_error`
- `invalid_token_contains_token_mismatch`
