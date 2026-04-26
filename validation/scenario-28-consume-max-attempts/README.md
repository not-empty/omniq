# Scenario 28: Consume Context Max Attempts

## Purpose

Validate that each SDK exposes `max_attempts` on the handler context and that
the value remains consistent across retries until the final successful attempt.

## Validate

- handler context includes `attempt`
- handler context includes `max_attempts`
- `max_attempts` matches the configured publish value
- retries expose increasing `attempt` values
- the handler can detect the last attempt using `attempt >= max_attempts`
- the job completes successfully on the last attempt

## Expected Outcome

- the handler sees three executions for a job published with `max_attempts = 3`
- the observed attempt sequence is `1`, `2`, `3`
- the observed `max_attempts` value is always `3`
- `is_last_attempt` is `false`, `false`, then `true`
- the final job state is `completed`

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- `python/run.py`
- `node/run.ts`
- `go/run.go`

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s28-python python /workspace/omniq/validation/scenario-28-consume-max-attempts/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s28-node npx tsx /workspace/omniq/validation/scenario-28-consume-max-attempts/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/bin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-28-consume-max-attempts/go && QUEUE=validation-s28-go /usr/bin/go run ."
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `job_id`
- `seen`
- `final_state`
