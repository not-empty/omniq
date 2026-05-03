# Scenario 32: Transport Backend Smoke

## Purpose

Validate that the same SDK transport path works under both standalone Redis and Redis Cluster.

## Validate

- client creation against the selected backend
- `publish`
- `reserve`
- queue discovery through `scan_queues()`

## Expected Outcome

- standalone Redis works through fallback-capable transports
- Redis Cluster works through cluster-capable transports
- the selected backend remains transparent to the contract surface

## Runners

- `python/run.py`
- `node/run.ts`
- `go/run.go`
- `php/run.php`
