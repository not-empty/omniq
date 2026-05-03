# Scenario 26: Consumer Drain True

## Purpose

Validate the observable `drain=true` consumer behavior across SDKs.

## Validate

- a running job is allowed to finish after `SIGINT`
- the finished job is acked normally
- the consumer exits after draining the current job
- no new job is reserved after the stop request

## Expected Outcome

- first job ends in `completed`
- second job remains in `wait`
- `completed_kept=1`
- `waiting=1`
- `active=0`

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](C:/Users/disarli/Documents/ops/omniq/validation/scenario-26-consume-drain-true/python/run.py)
- [node/run.ts](C:/Users/disarli/Documents/ops/omniq/validation/scenario-26-consume-drain-true/node/run.ts)
- [go/run.go](C:/Users/disarli/Documents/ops/omniq/validation/scenario-26-consume-drain-true/go/run.go)
- [php/run.php](/home/disarli/Downloads/omniq-core/validation/scenario-26-consume-drain-true/php/run.php)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s26-python python /workspace/omniq/validation/scenario-26-consume-drain-true/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s26-node npx tsx /workspace/omniq/validation/scenario-26-consume-drain-true/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/go/bin:/usr/bin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-26-consume-drain-true/go && /usr/local/go/bin/go mod tidy >/dev/null 2>&1 && QUEUE=validation-s26-go /usr/local/go/bin/go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && QUEUE=validation-s26-php php /workspace/omniq/validation/scenario-26-consume-drain-true/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `handled_job_ids`
- `first_job_state`
- `second_job_state`
- `stats`
