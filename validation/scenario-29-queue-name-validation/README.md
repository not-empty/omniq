# Scenario 29: Queue Name Validation

## Purpose

Validate that all SDKs reject invalid queue names consistently before queue operations run.

## Validate

- `publish` rejects invalid queue names
- monitor `stats` rejects invalid queue names
- a valid queue name still works

## Expected Outcome

- invalid queue names are rejected consistently
- manual hash tags are not accepted
- whitespace and illegal separators are rejected
- a valid queue publishes normally

## Runners

- `python/run.py`
- `node/run.ts`
- `go/run.go`
- `php/run.php`
