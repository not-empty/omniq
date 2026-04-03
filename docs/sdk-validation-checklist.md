# OmniQ SDK Validation Checklist

This document is the shared validation checklist for the OmniQ SDKs:

- Python
- Node
- Go

The goal is to verify that all SDKs:

- respect the core contract
- expose equivalent behavior
- interpret Redis state the same way
- stay aligned with the Lua source of truth

This checklist should be executed from the unified Docker environment in the core repository.

---

## Environment

From the core repository:

```bash
docker compose up -d
docker compose ps
```

Main services:

- `omniq-redis`
- `omniq-python`
- `omniq-node`
- `omniq-go`

Redis hostname inside containers:

- `omniq-redis`
- port `6379`

Useful shell entrypoints:

```bash
docker compose exec omniq-python sh
docker compose exec omniq-node sh
docker compose exec omniq-go sh
```

---

## Validation Rules

For every scenario below:

1. Run the same logical flow in Python, Node, and Go.
2. Validate both SDK return values and Redis state.
3. Compare the result against the contract in [omni-contract.md](/Users/disarli/Documents/ops/omniq/docs/omni-contract.md).
4. Compare monitoring expectations against [omniq_redis_map.md](/Users/disarli/Documents/ops/omniq/docs/omniq_redis_map.md).
5. Record any cross-SDK drift before changing code.

Recommended practice:

- use a fresh queue name per run, like `validation-basic-001`
- prefer deterministic job payloads and explicit job ids when helpful
- inspect Redis after each important transition
- keep Python as the matrix/reference behavior when SDK wrappers differ

---

## Scenario 1: Basic Publish And Reserve

Validate:

- publish accepts structured payloads only
- job enters `wait`
- reserve returns `JOB`
- reserve returns `lease_token`
- returned payload matches original JSON
- attempt starts correctly

Check:

- `{Q}:wait`
- `{Q}:job:{job_id}`
- `{Q}:active`
- `{Q}:idx:wait`
- `{Q}:idx:active`
- `{Q}:stats`

Expected:

- publish creates job hash and queue registration
- reserve moves job from wait to active
- `lease_token` is present in response and job hash
- `active` and `waiting_total` counters are updated consistently

---

## Scenario 2: Heartbeat Extends Lease

Validate:

- heartbeat succeeds only with the current `lease_token`
- `lock_until_ms` increases
- active zset score is updated
- invalid token is rejected

Check:

- `{Q}:job:{job_id}`
- `{Q}:active`

Expected:

- valid heartbeat returns the new `lock_until_ms`
- invalid token returns `TOKEN_MISMATCH`

---

## Scenario 3: Ack Success

Validate:

- success ack requires valid `lease_token`
- job transitions from `active` to `completed`
- lease fields are cleared
- completed retention/index updates work

Check:

- `{Q}:active`
- `{Q}:completed`
- `{Q}:idx:active`
- `{Q}:idx:completed`
- `{Q}:stats`

Expected:

- `active` decrements
- `completed_kept` reflects retained jobs
- job state becomes `completed`

---

## Scenario 4: Ack Fail With Retry

Validate:

- failure ack requires valid `lease_token`
- retryable job moves to `delayed`
- due time is computed from `backoff_ms`
- last error fields are stored

Check:

- `{Q}:job:{job_id}`
- `{Q}:delayed`
- `{Q}:idx:delayed`
- `{Q}:stats`

Expected:

- return is `("RETRY", due_ms)` or SDK equivalent
- state becomes `delayed`
- `active` decrements and `delayed` increments

---

## Scenario 5: Ack Fail To Terminal Failure

Validate:

- job reaches `failed` when attempts are exhausted
- no delayed reschedule happens
- failure indexes and counters are updated

Check:

- `{Q}:failed`
- `{Q}:idx:failed`
- `{Q}:stats`

Expected:

- return is `("FAILED", null)` or SDK equivalent
- state becomes `failed`

---

## Scenario 6: Promote Delayed

Validate:

- due delayed jobs are promoted back to runnable state
- destination lane is correct for grouped vs ungrouped jobs
- delayed indexes and counters are updated

Check:

- `{Q}:delayed`
- `{Q}:wait`
- `{Q}:g:{gid}:wait`
- `{Q}:idx:delayed`
- `{Q}:idx:wait`
- `{Q}:stats`

Expected:

- promoted jobs leave delayed
- runnable counters increase correctly

---

## Scenario 7: Reap Expired

Validate:

- expired active jobs are reaped
- retryable jobs move to delayed
- exhausted jobs move to failed
- orphaned active entries are cleaned up

Check:

- `{Q}:active`
- `{Q}:delayed`
- `{Q}:failed`
- lane indexes
- stats counters

Expected:

- Redis state matches retry or terminal behavior from the contract

---

## Scenario 8: Pause And Resume

Validate:

- pause creates the paused flag only
- reserve returns `PAUSED`
- running jobs are not moved
- resume removes the paused flag

Check:

- `{Q}:paused`
- active jobs before and after pause
- reserve result while paused

Expected:

- no lane movement caused by pause itself
- only new reserves are blocked

---

## Scenario 9: Grouped Jobs

Validate:

- grouped jobs go to group wait lanes
- `group_limit` initializes only when missing
- FIFO is preserved inside a group
- concurrency is limited per group
- grouped and ungrouped jobs both reserve correctly

Check:

- `{Q}:g:{gid}:wait`
- `{Q}:g:{gid}:inflight`
- `{Q}:g:{gid}:limit`
- `{Q}:groups:ready`
- `{Q}:stats`

Expected:

- inflight never exceeds group limit
- ready-group behavior matches the Redis map

---

## Scenario 10: Retry Failed And Remove Operations

Validate:

- `retry_failed` works only from `failed`
- batch retry respects per-job results
- remove rejects active jobs
- remove requires lane/state consistency
- batch remove keeps indexes and counters aligned

Check:

- lane structures
- `idx:*` keys
- stats counters
- job hash deletion

Expected:

- all management operations are atomic and lane-safe

---

## Scenario 11: Child Workflow Primitive

Validate:

- `childs_init` creates the shared completion counter
- `child_ack` is idempotent per child id
- counter reaches zero exactly once
- cleanup happens when count reaches zero

Check:

- `{base}:count`
- `{base}:done`

Expected:

- duplicate child ack does not decrement twice
- final completion returns zero

---

## Scenario 12: Monitor Queue Discovery And Stats

Validate:

- monitor lists queues from `omniq:queues`
- queue names are normalized consistently
- stats fields match actual Redis data
- paused status is derived correctly

SDK methods:

- `list_queues`
- `stats`
- `stats_many`

Expected:

- same queue list and same numeric values in Python, Node, and Go

---

## Scenario 13: Monitor Group Views

Validate:

- `groups_ready`
- `groups_ready_with_scores`
- `group_status`

Expected:

- same gids in the same order
- same score values
- same `ready`, `inflight`, `limit`, and `waiting_count`

Special focus:

- queues with more ready groups than the requested page size
- groups with explicit limits
- groups with inflight > 0

---

## Scenario 14: Monitor Lane Views

Validate:

- `lane_page`
- `find_jobs`
- `get_job`
- `overview`

Expected:

- same job ids
- same idx scores
- same state/timing fields
- overview samples match lane queries

Special focus:

- missing job hashes should be skipped consistently
- reverse ordering should behave the same

---

## Scenario 15: SDK Error Surface Parity

Validate:

- contract errors are surfaced consistently:
  - `TOKEN_MISMATCH`
  - `NOT_ACTIVE`
  - invalid batch limits
  - invalid publish payload types

Expected:

- each SDK may format exceptions differently
- but the underlying contract reason must remain visible and equivalent

---

## Sign-Off Criteria

The branch is ready only when:

- Lua scripts are synchronized across all repos
- the core contract matches the implemented behavior
- Python, Node, and Go produce equivalent results for all scenarios above
- monitor outputs match Redis truth
- no SDK-specific drift remains unexplained

---

## Result Template

Use a simple result log per run:

```text
Scenario:
Queue:
SDK:
Result:
Redis check:
Notes:
```

