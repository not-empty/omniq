# Scenario 11: Child Workflow Primitive

## Purpose

Validate the fan-out completion primitive.

## Validate

- `childs_init` creates the remaining counter
- `child_ack` is idempotent
- duplicate child ids do not decrement twice
- cleanup happens when the final child finishes

## Redis Checks

- `{base}:count`
- `{base}:done`

## Expected Outcome

- `childs_init(expected=3)` creates the counter
- first `child_ack("a")` returns `2`
- duplicate `child_ack("a")` also returns `2`
- `child_ack("b")` returns `1`
- final `child_ack("c")` returns `0`
- after the final child, both Redis keys are deleted

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-11-child-workflow/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-11-child-workflow/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-11-child-workflow/go/run.go)
- [php/run.php](/home/disarli/Downloads/omniq-core/validation/scenario-11-child-workflow/php/run.php)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'KEY=validation-s11-python python /workspace/omniq/validation/scenario-11-child-workflow/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'KEY=validation-s11-node npx tsx /workspace/omniq/validation/scenario-11-child-workflow/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-11-child-workflow/go && go mod tidy && KEY=validation-s11-go go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && KEY=validation-s11-php php /workspace/omniq/validation/scenario-11-child-workflow/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `key`
- `ack_sequence`
- `count_exists_after`
- `done_exists_after`
