# Scenario 03: Ack Success

## Purpose

Validate successful completion of an active job.

## Validate

- success ack requires valid `lease_token`
- job leaves active and enters completed
- completed retention and indexes are updated
- stats reflect the state transition

## Redis Checks

- `{Q}:active`
- `{Q}:completed`
- `{Q}:idx:active`
- `{Q}:idx:completed`
- `{Q}:stats`

## Expected Outcome

- active entry removed
- job state becomes `completed`
- `lease_token` and `lock_until_ms` are cleared
- completed history contains the job

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-03-ack-success/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-03-ack-success/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-03-ack-success/go/run.go)
- [php/run.php](/Users/disarli/Documents/ops/omniq/validation/scenario-03-ack-success/php/run.php)

Each runner validates:

- reserve returns a lease token
- ack success completes the job
- invalid token ack fails with a contract-visible reason

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s03-python python /workspace/omniq/validation/scenario-03-ack-success/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s03-node npx tsx /workspace/omniq/validation/scenario-03-ack-success/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-03-ack-success/go && go mod tidy && QUEUE=validation-s03-go go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && QUEUE=validation-s03-php php /workspace/omniq/validation/scenario-03-ack-success/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `job_id`
- `ack_success_ok`
- `invalid_token_error`
- `invalid_token_contains_token_mismatch`
