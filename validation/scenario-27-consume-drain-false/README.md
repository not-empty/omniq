# Scenario 27: Consumer Drain False

## Purpose

Validate the observable `drain=false` consumer behavior across SDKs.

## Validate

- what happens to the current in-flight job after `SIGINT`
- whether the current job finishes or is interrupted
- whether a second waiting job stays unreserved

## Expected Outcome

- the consumer exits after the stop request
- no second job is reserved
- Redis truth reveals whether the current job was drained or interrupted

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](C:/Users/disarli/Documents/ops/omniq/validation/scenario-27-consume-drain-false/python/run.py)
- [node/run.ts](C:/Users/disarli/Documents/ops/omniq/validation/scenario-27-consume-drain-false/node/run.ts)
- [go/run.go](C:/Users/disarli/Documents/ops/omniq/validation/scenario-27-consume-drain-false/go/run.go)
- [php/run.php](/home/disarli/Downloads/omniq-core/validation/scenario-27-consume-drain-false/php/run.php)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s27-python python /workspace/omniq/validation/scenario-27-consume-drain-false/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s27-node npx tsx /workspace/omniq/validation/scenario-27-consume-drain-false/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/go/bin:/usr/bin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-27-consume-drain-false/go && /usr/local/go/bin/go mod tidy >/dev/null 2>&1 && QUEUE=validation-s27-go /usr/local/go/bin/go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && QUEUE=validation-s27-php php /workspace/omniq/validation/scenario-27-consume-drain-false/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `child_exit_code`
- `handler_started`
- `handler_done`
- `first_job_state`
- `second_job_state`
- `stats`
