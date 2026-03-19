# OmniQ Redis Map

This document describes the proposed Redis key map for **OmniQ v1** with two layers:

1. **Transactional data**: the source of truth used by queue execution.
2. **Monitoring data**: lightweight metadata maintained atomically by the same Lua scripts to support cheap queue discovery and future manager/frontend views.

The goal is to **preserve the current transactional contract** and add a **non-breaking monitoring layer**.

---

## Design goals

- Keep the current queue execution model and key contract intact.
- Avoid expensive Redis key discovery / scans for monitoring.
- Support cheap queue listing for a future OmniQ Manager UI.
- Avoid high-cardinality monitoring structures when groups can be very numerous.
- Keep monitoring counters updated inside the same Lua scripts that mutate transactional state.

---

# 1. Transactional Redis Map

These keys represent the operational source of truth for queue behavior.

## Queue base

Given a queue name:

- `Q = {queue_name}`

### Core queue keys

- `{Q}:wait`
  - **Type:** LIST
  - **Purpose:** ungrouped jobs ready to be reserved.

- `{Q}:active`
  - **Type:** ZSET
  - **Purpose:** leased/running jobs.
  - **Score:** lease expiration timestamp (or equivalent active lease score used today).

- `{Q}:delayed`
  - **Type:** ZSET
  - **Purpose:** delayed jobs waiting for promotion.
  - **Score:** due timestamp.

- `{Q}:failed`
  - **Type:** LIST
  - **Purpose:** terminally failed jobs retained in queue history.

- `{Q}:completed`
  - **Type:** LIST
  - **Purpose:** completed jobs retained in queue history.
  - **Retention note:** currently capped / trimmed, so this is **kept history**, not lifetime totals.

- `{Q}:paused`
  - **Type:** STRING / existence flag
  - **Purpose:** queue pause flag.
  - **Note:** status only, not a monitoring counter.

## Grouped execution keys

- `{Q}:groups:ready`
  - **Type:** ZSET
  - **Purpose:** groups eligible to dispatch a grouped job.
  - **Score:** scheduler ordering score used by the current implementation.

- `{Q}:g:{gid}:wait`
  - **Type:** LIST
  - **Purpose:** waiting jobs for a specific group.

- `{Q}:g:{gid}:inflight`
  - **Type:** STRING / INTEGER
  - **Purpose:** current grouped inflight count for the group.

- `{Q}:g:{gid}:limit`
  - **Type:** STRING / INTEGER
  - **Purpose:** concurrency limit for the group.

## Job storage

- `{Q}:job:{id}`
  - **Type:** HASH
  - **Purpose:** job payload and job metadata.
  - **Typical contents:**
    - payload/body
    - group id if present
    - attempts / retries data
    - timing fields
    - lease token / status-related fields as used by current scripts

## Child/parent completion primitives

The current implementation also includes completion helper primitives for child flows.
These should remain transactional and unchanged.

Typical related keys are derived from the completion/checking logic already present in Lua/client code.
If these primitives are used, they remain part of the transactional layer.

---

# 2. Monitoring Redis Map

These keys are **new**, **non-breaking**, and intended only for observability / listing / manager views.

They should be updated atomically inside the same Lua scripts that already mutate the transactional keys.

## Global monitoring registry

- `omniq:queues`
  - **Type:** SET
  - **Purpose:** registry of known queue names.
  - **Usage:** cheap queue discovery for monitor / UI.
  - **Write rule:** every enqueue or any queue-touching mutation can ensure membership via `SADD`.

## Per-queue monitoring stats

- `{Q}:stats`
  - **Type:** HASH
  - **Purpose:** cheap summary counters for queue listing and health overview.

### Recommended fields in `{Q}:stats`

- `waiting`
  - Number of ungrouped waiting jobs.

- `group_waiting`
  - Number of grouped waiting jobs across all groups.

- `waiting_total`
  - Total waiting jobs.
  - Formula target:
    - `waiting_total = waiting + group_waiting`

- `active`
  - Number of currently active / leased jobs.

- `delayed`
  - Number of delayed jobs.

- `failed`
  - Number of jobs currently retained in `{Q}:failed`.

- `completed_kept`
  - Number of jobs currently retained in `{Q}:completed`.
  - Important: this is **not** lifetime completed count.

- `groups_ready`
  - Number of groups currently represented in `{Q}:groups:ready`.

- `last_activity_ms`
  - Last meaningful queue activity timestamp.

### Optional timestamp fields

These are useful if desired, but not strictly required for v1:

- `last_enqueue_ms`
- `last_reserve_ms`
- `last_finish_ms`

### Explicitly excluded from monitoring v1

To keep memory and write cost low, the following are intentionally excluded:

- no paused counter (`{Q}:paused` already represents status)
- no sample lists
- no full registry of all group ids
- no per-group permanent stats by default

---

# 3. Why no permanent group registry in monitoring v1

A full key such as:

- `{Q}:groups` (SET of every gid)

was considered, but rejected for v1 because:

- some queues may use very high-cardinality grouping
- groups may map to `user_id`, `tenant_id`, or similar dimensions
- a permanent gid registry can become large and costly only for observability purposes
- the manager primarily needs **queue-level** monitoring first, not full group enumeration

Therefore, monitoring v1 keeps only:

- aggregate grouped waiting count: `group_waiting`
- aggregate ready-group count: `groups_ready`

If future group drill-down becomes necessary, better options are:

- on-demand expensive inspection
- a capped “hot groups” index
- external analytics / event aggregation outside Redis

---

# 4. Queue state transitions and monitoring updates

The monitoring layer must be updated in the same Lua state transitions to avoid drift.

## 4.1 Enqueue

### Ungrouped immediate job

Transactional effect:
- push job id into `{Q}:wait`
- create/update `{Q}:job:{id}`

Monitoring effect:
- `SADD omniq:queues {Q}`
- `HINCRBY {Q}:stats waiting 1`
- `HINCRBY {Q}:stats waiting_total 1`
- set `last_activity_ms`
- optionally set `last_enqueue_ms`

### Grouped immediate job

Transactional effect:
- push job id into `{Q}:g:{gid}:wait`
- possibly add group to `{Q}:groups:ready` depending on readiness rules
- create/update `{Q}:job:{id}`

Monitoring effect:
- `SADD omniq:queues {Q}`
- `HINCRBY {Q}:stats group_waiting 1`
- `HINCRBY {Q}:stats waiting_total 1`
- if group newly enters ready set, increment `groups_ready`
- set `last_activity_ms`
- optionally set `last_enqueue_ms`

### Delayed job

Transactional effect:
- add job id into `{Q}:delayed`
- create/update `{Q}:job:{id}`

Monitoring effect:
- `SADD omniq:queues {Q}`
- `HINCRBY {Q}:stats delayed 1`
- set `last_activity_ms`
- optionally set `last_enqueue_ms`

---

## 4.2 Reserve

### Reserve from ungrouped wait

Transactional effect:
- pop from `{Q}:wait`
- add to `{Q}:active`

Monitoring effect:
- `HINCRBY {Q}:stats waiting -1`
- `HINCRBY {Q}:stats waiting_total -1`
- `HINCRBY {Q}:stats active 1`
- set `last_activity_ms`
- optionally set `last_reserve_ms`

### Reserve from grouped wait

Transactional effect:
- pop from `{Q}:g:{gid}:wait`
- add to `{Q}:active`
- update inflight / readiness as needed

Monitoring effect:
- `HINCRBY {Q}:stats group_waiting -1`
- `HINCRBY {Q}:stats waiting_total -1`
- `HINCRBY {Q}:stats active 1`
- if group leaves ready set, decrement `groups_ready`
- set `last_activity_ms`
- optionally set `last_reserve_ms`

---

## 4.3 Ack success

Transactional effect:
- remove job from `{Q}:active`
- push job to `{Q}:completed`
- trim `{Q}:completed` to retention limit

Monitoring effect:
- `HINCRBY {Q}:stats active -1`
- adjust `completed_kept`
  - simplest/safest approach: recompute from retained length after trim
  - or increment then correct if trim removed excess entries
- set `last_activity_ms`
- optionally set `last_finish_ms`

Important note:
- because completed retention is capped, `completed_kept` represents current retained entries only.

---

## 4.4 Ack fail

### Retryable failure -> delayed

Transactional effect:
- remove from `{Q}:active`
- add to `{Q}:delayed`

Monitoring effect:
- `HINCRBY {Q}:stats active -1`
- `HINCRBY {Q}:stats delayed 1`
- set `last_activity_ms`
- optionally set `last_finish_ms`

### Terminal failure -> failed

Transactional effect:
- remove from `{Q}:active`
- push to `{Q}:failed`

Monitoring effect:
- `HINCRBY {Q}:stats active -1`
- `HINCRBY {Q}:stats failed 1`
- set `last_activity_ms`
- optionally set `last_finish_ms`

---

## 4.5 Promote delayed

Transactional effect:
- move due jobs from `{Q}:delayed` to either:
  - `{Q}:wait`, or
  - `{Q}:g:{gid}:wait`
- possibly add group into `{Q}:groups:ready`

Monitoring effect:
- for each promoted ungrouped job:
  - `HINCRBY {Q}:stats delayed -1`
  - `HINCRBY {Q}:stats waiting 1`
  - `HINCRBY {Q}:stats waiting_total 1`
- for each promoted grouped job:
  - `HINCRBY {Q}:stats delayed -1`
  - `HINCRBY {Q}:stats group_waiting 1`
  - `HINCRBY {Q}:stats waiting_total 1`
  - if group newly enters ready set, increment `groups_ready`
- set `last_activity_ms`

---

## 4.6 Reap expired

### Reaped job retried via delayed

Transactional effect:
- remove expired job from `{Q}:active`
- add to `{Q}:delayed`

Monitoring effect:
- `HINCRBY {Q}:stats active -1`
- `HINCRBY {Q}:stats delayed 1`
- set `last_activity_ms`

### Reaped job terminally failed

Transactional effect:
- remove expired job from `{Q}:active`
- push to `{Q}:failed`

Monitoring effect:
- `HINCRBY {Q}:stats active -1`
- `HINCRBY {Q}:stats failed 1`
- set `last_activity_ms`

---

## 4.7 Retry failed / retry failed batch

### Retry terminal failure back to runnable state

Transactional effect:
- remove job from `{Q}:failed`
- send to `{Q}:wait`, `{Q}:g:{gid}:wait`, or `{Q}:delayed` depending on behavior

Monitoring effect:
- `HINCRBY {Q}:stats failed -1`
- then increment one of:
  - `waiting` + `waiting_total`, or
  - `group_waiting` + `waiting_total`, or
  - `delayed`
- if grouped and newly ready, increment `groups_ready`
- set `last_activity_ms`

---

## 4.8 Remove job / remove jobs batch

Removal logic must update the stats according to the lane from which the job was removed.

Possible adjustments:
- from `{Q}:wait` -> decrement `waiting` and `waiting_total`
- from grouped wait -> decrement `group_waiting` and `waiting_total`
- from `{Q}:active` -> decrement `active`
- from `{Q}:delayed` -> decrement `delayed`
- from `{Q}:failed` -> decrement `failed`
- from `{Q}:completed` -> decrement `completed_kept` if retained entry is removed
- if grouped readiness changes as a result, adjust `groups_ready`
- set `last_activity_ms`

Because removal may target different states, the implementation must determine the actual source lane before adjusting counters.

---

# 5. Monitoring philosophy

## Transactional layer is the truth

The existing queue keys remain the authoritative state for execution.

## Monitoring layer is a cheap summary

The monitoring keys are only there to make these operations cheap:

- list queues
- show queue cards in a UI
- display counts quickly without multiple raw Redis reads per queue
- support basic queue health dashboards

## No contract break

Client behavior and current queue semantics remain unchanged.

## No high-cardinality observability by default

Monitoring v1 intentionally avoids storing every group id.

---

# 6. Recommended v1 implementation scope

## Add

- `omniq:queues`
- `{Q}:stats`

## Do not add yet

- permanent `{Q}:groups` gid registry
- per-group permanent stats hashes
- sample lists
- historical analytics in Redis

---

# 7. Example monitoring view payload

A future OmniQ Manager queue list could be powered from `{Q}:stats` like this:

```json
{
  "queue": "emails",
  "waiting": 120,
  "group_waiting": 4820,
  "waiting_total": 4940,
  "active": 73,
  "delayed": 19,
  "failed": 4,
  "completed_kept": 100,
  "groups_ready": 12,
  "paused": false,
  "last_activity_ms": 1773200000123
}
```

Note:
- `paused` can be derived from existence/value of `{Q}:paused`
- it does not need its own counter in monitoring

---

# 8. Final recommendation

For OmniQ monitoring v1:

- keep the current Redis execution model unchanged
- add a global queue registry
- add a per-queue stats hash
- maintain monitoring counters atomically in Lua
- avoid storing all group ids when grouping cardinality may be high

This gives OmniQ a strong foundation for a future Manager frontend without changing queue contracts or overloading Redis with observability-only structures.
