# Scenario 30: Scan Queues Discovery Rules

## Purpose

Validate the exact discovery semantics of `scan_queues()` and `stats_many()` under sparse monitor keys.

## Validate

- valid `*:stats` keys are discovered
- paused-only queues with no stats are not discovered
- malformed stats keys are ignored
- `stats_many()` with no explicit queue list follows the same discovery set

## Expected Outcome

- only valid queue names backed by stats are returned
- invalid sparse keys do not leak into discovery
- paused-only queues remain invisible to discovery

## Runners

- `python/run.py`
- `node/run.ts`
- `go/run.go`
- `php/run.php`
