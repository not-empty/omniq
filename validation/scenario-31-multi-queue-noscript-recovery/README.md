# Scenario 31: Multi-Queue NOSCRIPT Recovery

## Purpose

Validate `NOSCRIPT` recovery across more than one queue after repeated script flushes.

## Validate

- `publish` after repeated `SCRIPT FLUSH`
- `reserve` after repeated `SCRIPT FLUSH`
- `heartbeat` after repeated `SCRIPT FLUSH`
- `ack_success` after repeated `SCRIPT FLUSH`
- queue discovery still sees both queues

## Expected Outcome

- both queues complete successfully
- script cache misses recover automatically for every SDK
- both queues are still discoverable after the recovery flow

## Runners

- `python/run.py`
- `node/run.ts`
- `go/run.go`
- `php/run.php`
