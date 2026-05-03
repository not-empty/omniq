# Scenario 15: Error Surface Parity

## Purpose

Validate that contract failures are visible and equivalent across SDKs.

## Validate

- `TOKEN_MISMATCH`
- `NOT_ACTIVE`
- invalid batch-size errors
- invalid publish payload errors
- lane mismatch errors

## Expected Outcome

- exception or error formatting may differ by language
- contract reason must remain visible and recognizable

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-15-error-surface/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-15-error-surface/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-15-error-surface/go/run.go)
- [php/run.php](/home/disarli/Downloads/omniq-core/validation/scenario-15-error-surface/php/run.php)

Each runner validates visible error reasons for:

- bad-token ack
- second ack after job is no longer active
- oversized batch call
- invalid publish payload
- lane mismatch removal

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s15-python python /workspace/omniq/validation/scenario-15-error-surface/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s15-node npx tsx /workspace/omniq/validation/scenario-15-error-surface/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-15-error-surface/go && go mod tidy && QUEUE=validation-s15-go go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && QUEUE=validation-s15-php php /workspace/omniq/validation/scenario-15-error-surface/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `token_mismatch`
- `not_active`
- `batch_limit`
- `invalid_publish`
- `lane_mismatch`
