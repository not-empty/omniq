# Scenario 13: Monitor Group Views

## Purpose

Validate group monitoring APIs.

## Validate

- `groups_ready`
- `groups_ready_with_scores`
- `group_status`

## Special Focus

- ready-group pagination
- direct readiness truth
- inflight and limit values
- waiting counts

## Expected Outcome

- `groups_ready(limit=2)` returns only the first page of ready gids
- `groups_ready_with_scores(limit=10)` returns all ready gids with scores
- `group_status([...])` still reports `ready=true` for gids outside that first page
- explicit limit and inflight values are reflected correctly

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](/Users/disarli/Documents/ops/omniq/validation/scenario-13-monitor-groups/python/run.py)
- [node/run.ts](/Users/disarli/Documents/ops/omniq/validation/scenario-13-monitor-groups/node/run.ts)
- [go/run.go](/Users/disarli/Documents/ops/omniq/validation/scenario-13-monitor-groups/go/run.go)
- [php/run.php](/home/disarli/Downloads/omniq-core/validation/scenario-13-monitor-groups/php/run.php)

Each runner seeds:

- `alpha` with limit `2`, two jobs, and one active lease
- `beta`, `gamma`, and `delta` with one waiting job each

Then it validates:

- `groups_ready`
- `groups_ready_with_scores`
- `group_status`

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s13-python python /workspace/omniq/validation/scenario-13-monitor-groups/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s13-node npx tsx /workspace/omniq/validation/scenario-13-monitor-groups/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-13-monitor-groups/go && go mod tidy && QUEUE=validation-s13-go go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && QUEUE=validation-s13-php php /workspace/omniq/validation/scenario-13-monitor-groups/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `groups_ready_page`
- `groups_ready_all`
- `group_status`
