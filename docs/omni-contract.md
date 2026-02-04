# OmniQ v1 — Omni-language Client Contract

This document defines the **minimum, language-agnostic contract** that every OmniQ client MUST implement.

OmniQ v1 is Redis + Lua:
- Queue is **hybrid**: ungrouped by default, optional groups per job (FIFO per group + concurrency limit).
- Leases are **token-gated**: RESERVE returns a `lease_token` that MUST be used by HEARTBEAT and ACK.
- Pause / resume is **flag-only**: it prevents new reserves and does not move jobs.

This contract is intentionally strict so all language SDKs behave the same.

---

## Terms

- `queue`: logical queue name
- `base`: Redis hash-tagged key prefix: `{queue}` (keys share the same Redis slot)
- `job_id`: string (ULID recommended)
- `payload`: JSON-serialized string stored in Redis
- `gid`: optional group id (string). Empty means ungrouped lane
- `lease_token`: unique token created during RESERVE and stored on the job; gates ACK / HEARTBEAT

---

## Key behaviors (normative)

### Pause / resume
- Pause is **queue-level** and **flag-only**
- Pause does NOT move jobs between lanes
- Pause does NOT affect jobs already running
- Pause only blocks granting **new reserves**

A queue is paused when this key exists:
- `{base}:paused` (STRING) — value irrelevant; presence means paused

When paused:
- `RESERVE` MUST return `PAUSED`

---

### Leases + token gating
- `RESERVE` creates a lease and returns `(lock_until_ms, lease_token)`
- `HEARTBEAT` extends a lease ONLY if `lease_token` matches the job
- `ACK_SUCCESS` / `ACK_FAIL` ONLY succeed if `lease_token` matches

If a worker loses the lease or token mismatches, it MUST stop processing (best effort).

---

## Required client API

### Base helpers
1. `queue_base(queue: string) -> string`
   - returns `{queue}` if not already wrapped

2. `now_ms() -> int`
   - milliseconds since epoch

3. `ulid() -> string`
   - optional helper, but strongly recommended

---

## Publish

### Publish job (single canonical method)

```
publish(queue, payload, opts) -> job_id
```

#### Inputs
- `queue`: string
- `payload`: **JSON object or array only**
  - Client MUST serialize payload to JSON string
  - Passing a raw string is an error
- `opts` (optional):
  - `job_id` string (optional; client MAY generate one)
  - `timeout_ms` int
  - `max_attempts` int
  - `backoff_ms` int
  - `due_ms` int
  - `gid` string (optional; grouped job)
  - `group_limit` int (>0 initializes group limit if missing)

#### Output
- `job_id` (string)

Notes:
- There is intentionally **no separate low-level publish** in the contract
- All clients MUST serialize structured payloads consistently

---

## Reserve

```
reserve(queue, now_ms_override=0) -> ReserveResult
```

Return union:
- `null` → no job available
- `{ status: "PAUSED" }` → queue paused
- `{ status: "JOB", job_id, payload, lock_until_ms, attempt, gid, lease_token }`

Rules:
- `RESERVE` MUST return `PAUSED` when paused
- `lease_token` MUST be present on job responses

---

## Heartbeat

```
heartbeat(queue, job_id, lease_token, now_ms_override=0) -> lock_until_ms
```

Errors:
- MUST surface `NOT_ACTIVE`
- MUST surface `TOKEN_MISMATCH`

---

## Ack success

```
ack_success(queue, job_id, lease_token, now_ms_override=0) -> void
```

Errors:
- MUST surface `NOT_ACTIVE`
- MUST surface `TOKEN_MISMATCH`

---

## Ack fail

```
ack_fail(queue, job_id, lease_token, now_ms_override=0) -> AckFailResult
```

Return:
- `("RETRY", due_ms)`
- `("FAILED", null)`

Errors:
- MUST surface `NOT_ACTIVE`
- MUST surface `TOKEN_MISMATCH`

---

## Maintenance calls

1. `promote_delayed(queue, max_promote=1000, now_ms_override=0) -> int`
2. `reap_expired(queue, max_reap=1000, now_ms_override=0) -> int`

---

## Pause / resume

1. `pause(queue) -> string`
2. `resume(queue) -> int`
3. `is_paused(queue) -> bool` (optional helper)

---

## Consumer helper (optional)

A client MAY provide a `consume()` helper that:
- periodically calls `promote_delayed` and `reap_expired`
- calls `reserve`
- runs `handler(ctx)`
- heartbeats while handler runs
- ACKs success or failure using `lease_token`

### Shutdown semantics (recommended)

Clients SHOULD support:
- **drain=true**: finish current job, then exit
- **drain=false**: stop reserving immediately; exit ASAP

Exact signal handling is runtime-specific and not mandated by the contract.

---

## Job context (handler input)

Handlers receive:

- `queue`
- `job_id`
- `payload_raw` (JSON string)
- `payload` (parsed JSON object / array)
- `attempt` (int)
- `lock_until_ms` (int)
- `lease_token` (string)
- `gid` (string)
