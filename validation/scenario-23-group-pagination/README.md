# Scenario 23: Group Pagination Stress

## Purpose

Validate stable pagination for ready groups and direct readiness truth for gids outside the current page.

## Validate

- `groups_ready(offset, limit)` across multiple pages
- `groups_ready_with_scores(offset, limit)` across multiple pages
- `group_status([...])` for gids outside the first page

## Expected Outcome

- ready group pages preserve zset order
- scored pages preserve the same order and score values
- `group_status` reports `ready=true` even for gids not present in the first page
- limit, inflight, and waiting_count values remain consistent

## Runners

This scenario has one tiny runner per SDK, all owned by the contract repo:

- [python/run.py](C:/Users/disarli/Documents/ops/omniq/validation/scenario-23-group-pagination/python/run.py)
- [node/run.ts](C:/Users/disarli/Documents/ops/omniq/validation/scenario-23-group-pagination/node/run.ts)
- [go/run.go](C:/Users/disarli/Documents/ops/omniq/validation/scenario-23-group-pagination/go/run.go)
- [php/run.php](/home/disarli/Downloads/omniq-core/validation/scenario-23-group-pagination/php/run.php)

## Suggested Commands

Python:

```bash
docker compose exec omniq-python sh -lc 'QUEUE=validation-s23-python python /workspace/omniq/validation/scenario-23-group-pagination/python/run.py'
```

Node:

```bash
docker compose exec omniq-node sh -lc 'QUEUE=validation-s23-node npx tsx /workspace/omniq/validation/scenario-23-group-pagination/node/run.ts'
```

Go:

```bash
docker compose exec omniq-go sh -lc "export PATH=/usr/local/go/bin:/usr/bin:/bin; export GOTOOLCHAIN=auto; cd /workspace/omniq/validation/scenario-23-group-pagination/go && go mod tidy >/dev/null 2>&1 && QUEUE=validation-s23-go /usr/local/go/bin/go run ."
```

PHP:

```bash
docker compose exec omniq-php sh -lc 'cd /workspace/omniq-php && QUEUE=validation-s23-php php /workspace/omniq/validation/scenario-23-group-pagination/php/run.php'
```

## Output Shape

Each runner prints a JSON object with:

- `sdk`
- `queue`
- `groups_ready_page_1`
- `groups_ready_page_2`
- `groups_ready_scored_page_1`
- `groups_ready_scored_page_2`
- `group_status`
- `groups_ready_raw`
