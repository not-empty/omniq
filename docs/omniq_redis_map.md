# OmniQ Redis Map

This document describes the **current Redis key map implemented by OmniQ**.

It reflects the latest Lua scripts and includes three practical layers:

1. **Transactional data**: the source of truth used by queue execution.
2. **Operational indexes**: lightweight secondary indexes used for management, lookup, and efficient lane inspection.
3. **Monitoring data**: lightweight queue-level metadata maintained atomically by the same Lua scripts.

The design keeps queue execution authoritative in the transactional keys while exposing cheap counters and indexes for manager, monitor, and maintenance operations.

---

## Design goals

- Keep queue execution deterministic and Redis-native.
- Preserve the transactional model as the source of truth.
- Support cheap queue discovery without broad Redis scans.
- Support queue management operations such as remove, retry, and lane inspection.
- Avoid high-cardinality observability structures where they are not needed.
- Maintain counters and indexes inside the same Lua scripts that mutate transactional state.
- Keep grouped execution fair and cluster-safe.

---

# 1. Queue base and anchor model

Given a queue name:

- `Q = {queue_name}`

OmniQ uses a queue **base key** and a queue **anchor key**:

- **Base:** `{Q}`
- **Anchor:** `{Q}:meta`

Lua scripts receive the anchor in `KEYS[1]` and derive the base internally. This preserves cluster slot affinity while keeping key derivation consistent across scripts.

---

# 2. Transactional Redis Map

These keys are the operational source of truth for queue behavior.

## 2.1 Core queue lanes

- `{Q}:wait`
  - **Type:** LIST
  - **Purpose:** ungrouped runnable jobs waiting to be reserved.

- `{Q}:active`
  - **Type:** ZSET
  - **Purpose:** leased/running jobs.
  - **Score:** `lock_until_ms` lease expiration timestamp.

- `{Q}:delayed`
  - **Type:** ZSET
  - **Purpose:** delayed jobs waiting for promotion.
  - **Score:** `due_ms`.

- `{Q}:failed`
  - **Type:** LIST
  - **Purpose:** terminally failed jobs retained in queue history.

- `{Q}:completed`
  - **Type:** LIST
  - **Purpose:** completed jobs retained in queue history.
  - **Retention:** capped to the configured keep limit in Lua.
  - **Current implementation:** capped to 100 kept jobs in `ack_success.lua`.

- `{Q}:paused`
  - **Type:** STRING / existence flag
  - **Purpose:** queue pause flag.
  - **Semantics:** when present, reserve returns `PAUSED` and new reserve operations do not dispatch work.

## 2.2 Grouped execution keys

- `{Q}:groups:ready`
  - **Type:** ZSET
  - **Purpose:** groups currently eligible to dispatch a grouped job.
  - **Score:** scheduling score used by grouped fairness logic.

- `{Q}:g:{gid}:wait`
  - **Type:** LIST
  - **Purpose:** runnable waiting jobs for a specific group.

- `{Q}:g:{gid}:inflight`
  - **Type:** STRING / INTEGER
  - **Purpose:** current grouped inflight count for the group.

- `{Q}:g:{gid}:limit`
  - **Type:** STRING / INTEGER
  - **Purpose:** per-group concurrency limit.
  - **Default:** `1` when missing or invalid.

- `{Q}:has_groups`
  - **Type:** STRING / existence/value flag
  - **Purpose:** indicates that the queue has seen grouped jobs.
  - **Current use:** set during grouped enqueue. It is informational and can support manager/inspection behavior.

## 2.3 Scheduler helper keys

- `{Q}:lane:rr`
  - **Type:** STRING / INTEGER
  - **Purpose:** round-robin toggle between grouped and ungrouped reservation attempts.
  - **Semantics:**
    - `0` => try grouped first, then ungrouped
    - `1` => try ungrouped first, then grouped

- `{Q}:lease:seq`
  - **Type:** STRING / INTEGER
  - **Purpose:** monotonic sequence used to generate unique lease tokens.

## 2.4 Job storage

- `{Q}:job:{job_id}`
  - **Type:** HASH
  - **Purpose:** canonical job payload and job metadata.

### Current job hash fields

Common fields written by the current Lua implementation:

- `id`
- `payload`
- `gid` when grouped
- `state`
- `attempt`
- `max_attempts`
- `timeout_ms`
- `backoff_ms`
- `created_ms`
- `updated_ms`
- `queued_ms`
- `first_started_ms`
- `last_started_ms`
- `completed_ms`
- `failed_ms`
- `due_ms` when delayed
- `lock_until_ms` when leased
- `lease_token` when leased
- `last_error`
- `last_error_ms`

### State model

A job may move through these states:

- `wait`
- `active`
- `delayed`
- `failed`
- `completed`

## 2.5 Child completion primitives

OmniQ also includes transactional helper primitives for child/parent completion flows.

- `{base}:count`
  - **Type:** STRING / INTEGER
  - **Purpose:** remaining child acknowledgements expected.

- `{base}:done`
  - **Type:** SET
  - **Purpose:** deduplication set of child ids already acknowledged.

These are initialized by `childs_init.lua` and consumed by `child_ack.lua`.

---

# 3. Operational index Redis Map

These keys are secondary indexes. They do not replace transactional truth, but they make management and inspection cheap and predictable.

## 3.1 Per-lane job indexes

- `{Q}:idx:wait`
  - **Type:** ZSET
  - **Purpose:** index of jobs currently in runnable waiting lanes.
  - **Members:** job ids from both:
    - `{Q}:wait`
    - `{Q}:g:{gid}:wait`
  - **Score:** index insertion/update time, usually `now_ms`.

- `{Q}:idx:active`
  - **Type:** ZSET
  - **Purpose:** index of jobs currently active.
  - **Score:** activation/index time, usually `now_ms`.

- `{Q}:idx:delayed`
  - **Type:** ZSET
  - **Purpose:** index of jobs currently delayed.
  - **Score:** time when the job entered the delayed lane, usually `now_ms`.
  - **Note:** this is distinct from `{Q}:delayed`, whose score is the actual `due_ms`.

- `{Q}:idx:failed`
  - **Type:** ZSET
  - **Purpose:** index of jobs currently retained in failed state.
  - **Score:** time when the job entered failed state.

- `{Q}:idx:completed`
  - **Type:** ZSET
  - **Purpose:** index of jobs currently retained in completed state.
  - **Score:** time when the job entered completed state.

## 3.2 Why the indexes exist

These indexes support management operations without having to inspect every lane structure directly.

Typical uses:

- list jobs by lane in manager screens
- paginate or filter lane contents cheaply
- remove jobs from known lanes
- retry failed jobs after selection
- correlate retained completed/failed jobs with current history windows

## 3.3 Truth versus index

The lane structures remain authoritative:

- LIST/ZSET queue lanes and the job hash are the source of truth
- `idx:*` keys are operational mirrors maintained atomically by Lua

If an index ever drifts, the transactional lane still defines the real state.

---

# 4. Monitoring Redis Map

These keys are queue-level monitoring structures maintained atomically by the same Lua scripts.

## 4.1 Queue discovery

Queue discovery is no longer maintained as a global Redis registry.

The monitoring/discovery model is:

- scan queue-local `*:stats` hashes
- normalize the discovered `{Q}` base back to queue name
- treat discovery as an operational/admin concern, not a transactional Redis invariant

This separation keeps queue execution atomic while allowing cluster-safe queue discovery.

## 4.2 Per-queue stats

- `{Q}:stats`
  - **Type:** HASH
  - **Purpose:** cheap summary counters and timestamps for queue listing, health calculation, and dashboards.

### Current stats fields

- `waiting`
  - Number of ungrouped waiting jobs.

- `group_waiting`
  - Number of grouped waiting jobs across all group wait lists.

- `waiting_total`
  - Total waiting jobs.
  - Intended invariant:
    - `waiting_total = waiting + group_waiting`

- `active`
  - Number of currently leased jobs.

- `delayed`
  - Number of delayed jobs.

- `failed`
  - Number of jobs currently retained in `{Q}:failed`.

- `completed_kept`
  - Number of jobs currently retained in `{Q}:completed`.
  - This is retained history only, not lifetime completions.

- `groups_ready`
  - Number of groups currently present in `{Q}:groups:ready`.

- `last_activity_ms`
  - Last meaningful queue mutation timestamp.

- `last_enqueue_ms`
  - Timestamp of most recent enqueue.

- `last_reserve_ms`
  - Timestamp of most recent successful reserve.

- `last_finish_ms`
  - Timestamp of most recent terminal finish action from active state.
  - Updated by success and fail acknowledgements.

## 4.3 Explicitly excluded from stats

To keep memory and write cost low, the current model intentionally excludes:

- paused counters
- permanent group registries
- per-group stats hashes
- historical timeseries in Redis
- lifetime completed totals

Paused status is derived directly from existence of `{Q}:paused`.

---

# 5. Queue state transitions and exact key effects

The monitoring and index layers are updated in the same Lua state transitions to avoid drift.

## 5.1 Enqueue (`enqueue.lua`)

### Ungrouped immediate enqueue

Transactional effect:
- create `{Q}:job:{job_id}`
- `RPUSH` into `{Q}:wait`

Index effect:
- `ZADD {Q}:idx:wait now_ms job_id`

Monitoring effect:
- `HINCRBY {Q}:stats waiting 1`
- `HINCRBY {Q}:stats waiting_total 1`
- set `last_activity_ms`
- set `last_enqueue_ms`

### Grouped immediate enqueue

Transactional effect:
- create `{Q}:job:{job_id}`
- set `{Q}:has_groups = 1`
- optionally initialize `{Q}:g:{gid}:limit`
- `RPUSH` into `{Q}:g:{gid}:wait`
- possibly `ZADD` gid into `{Q}:groups:ready`

Index effect:
- `ZADD {Q}:idx:wait now_ms job_id`

Monitoring effect:
- `HINCRBY {Q}:stats group_waiting 1`
- `HINCRBY {Q}:stats waiting_total 1`
- increment `groups_ready` if the group is newly made ready
- set `last_activity_ms`
- set `last_enqueue_ms`

### Delayed enqueue

Transactional effect:
- create `{Q}:job:{job_id}`
- `ZADD {Q}:delayed due_ms job_id`
- set job state to `delayed`

Index effect:
- `ZADD {Q}:idx:delayed now_ms job_id`

Monitoring effect:
- `HINCRBY {Q}:stats delayed 1`
- set `last_activity_ms`
- set `last_enqueue_ms`

## 5.2 Reserve (`reserve.lua`)

### Common reserve behavior

Transactional effect:
- if `{Q}:paused` exists, return `PAUSED`
- try grouped and ungrouped lanes in round-robin order using `{Q}:lane:rr`
- generate a unique lease token using `{Q}:lease:seq`
- mark the selected job as `active`
- increment `attempt`
- set `lock_until_ms`
- set `lease_token`
- set `last_started_ms`
- set `first_started_ms` only on the first lease
- add job to `{Q}:active` with score `lock_until_ms`

Index effect:
- `ZREM {Q}:idx:wait job_id`
- `ZADD {Q}:idx:active now_ms job_id`

Monitoring effect:
- decrement either `waiting` or `group_waiting`
- decrement `waiting_total`
- increment `active`
- decrement `groups_ready` when a gid is popped from `{Q}:groups:ready`
- possibly re-add the group to `{Q}:groups:ready` if more grouped jobs remain and inflight stays below limit
- set `last_activity_ms`
- set `last_reserve_ms`

## 5.3 Heartbeat (`heartbeat.lua`)

Transactional effect:
- requires matching `lease_token`
- extends `lock_until_ms`
- updates the score of `{Q}:active`

Index effect:
- none on `idx:*`

Monitoring effect:
- none on `{Q}:stats`

## 5.4 Ack success (`ack_success.lua`)

Transactional effect:
- requires matching `lease_token`
- `ZREM` from `{Q}:active`
- mark job `completed`
- set `completed_ms`
- clear `lease_token`
- clear `lock_until_ms`
- `LPUSH` into `{Q}:completed`
- trim retained completed list to keep limit
- for grouped jobs, decrement `{Q}:g:{gid}:inflight`
- possibly re-add gid into `{Q}:groups:ready`

Index effect:
- `ZREM {Q}:idx:active job_id`
- `ZADD {Q}:idx:completed now_ms job_id`
- when retained completed jobs are trimmed, remove old ids from `{Q}:idx:completed`
- when an old retained completed job is trimmed, its `{Q}:job:{old_id}` hash is deleted

Monitoring effect:
- decrement `active`
- update `completed_kept` from retained completed length
- possibly increment `groups_ready`
- set `last_activity_ms`
- set `last_finish_ms`

## 5.5 Ack fail (`ack_fail.lua`)

Transactional effect:
- requires matching `lease_token`
- `ZREM` from `{Q}:active`
- optionally store `last_error` and `last_error_ms`
- for grouped jobs, decrement `{Q}:g:{gid}:inflight`
- possibly re-add gid into `{Q}:groups:ready`

### Retryable fail path

Transactional effect:
- set state `delayed`
- set `due_ms = now_ms + backoff_ms`
- clear `lease_token`
- clear `lock_until_ms`
- `ZADD {Q}:delayed due_ms job_id`

Index effect:
- `ZREM {Q}:idx:active job_id`
- `ZADD {Q}:idx:delayed now_ms job_id`

Monitoring effect:
- decrement `active`
- increment `delayed`
- possibly increment `groups_ready`
- set `last_activity_ms`
- set `last_finish_ms`

### Terminal fail path

Transactional effect:
- set state `failed`
- set `failed_ms`
- clear `lease_token`
- clear `lock_until_ms`
- `LPUSH {Q}:failed job_id`

Index effect:
- `ZREM {Q}:idx:active job_id`
- `ZADD {Q}:idx:failed now_ms job_id`

Monitoring effect:
- decrement `active`
- increment `failed`
- possibly increment `groups_ready`
- set `last_activity_ms`
- set `last_finish_ms`

## 5.6 Promote delayed (`promote_delayed.lua`)

Transactional effect:
- move due jobs from `{Q}:delayed` back to runnable state
- destination is either:
  - `{Q}:wait`, or
  - `{Q}:g:{gid}:wait`
- grouped promotions may add gid into `{Q}:groups:ready`

Index effect:
- `ZREM {Q}:idx:delayed job_id`
- `ZADD {Q}:idx:wait now_ms job_id`

Monitoring effect:
- decrement `delayed`
- increment either `waiting` or `group_waiting`
- increment `waiting_total`
- increment `groups_ready` when a group newly becomes ready
- set `last_activity_ms`

## 5.7 Reap expired (`reap_expired.lua`)

Transactional effect:
- find expired jobs in `{Q}:active` by `lock_until_ms <= now_ms`
- `ZREM` expired jobs from `{Q}:active`
- clear stale `lease_token`
- for grouped jobs, decrement group inflight and possibly re-add group readiness

### Reap to delayed

Transactional effect:
- set state `delayed`
- set `due_ms = now_ms + backoff_ms`
- clear `lock_until_ms`
- `ZADD {Q}:delayed due_ms job_id`

Index effect:
- `ZREM {Q}:idx:active job_id`
- `ZADD {Q}:idx:delayed now_ms job_id`

Monitoring effect:
- decrement `active`
- increment `delayed`
- possibly increment `groups_ready`
- set `last_activity_ms`

### Reap to failed

Transactional effect:
- set state `failed`
- set `failed_ms`
- clear `lock_until_ms`
- `LPUSH {Q}:failed job_id`

Index effect:
- `ZREM {Q}:idx:active job_id`
- `ZADD {Q}:idx:failed now_ms job_id`

Monitoring effect:
- decrement `active`
- increment `failed`
- possibly increment `groups_ready`
- set `last_activity_ms`

### Reap of orphaned active entry

If the job hash no longer exists but the active zset member does exist:

Transactional effect:
- remove the active member only

Index effect:
- `ZREM {Q}:idx:active job_id`

Monitoring effect:
- decrement `active`
- set `last_activity_ms`

## 5.8 Retry failed (`retry_failed.lua`)

Transactional effect:
- only valid for jobs in state `failed`
- remove stale occurrences from failed/wait/delayed lane structures
- clear `due_ms`
- clear `failed_ms`
- clear `lease_token`
- clear `lock_until_ms`
- reset `attempt = 0`
- set state back to `wait`
- push back into either:
  - `{Q}:wait`, or
  - `{Q}:g:{gid}:wait`
- grouped retries may add gid into `{Q}:groups:ready`

Index effect:
- `ZREM {Q}:idx:failed job_id`
- `ZREM {Q}:idx:delayed job_id`
- `ZADD {Q}:idx:wait now_ms job_id`

Monitoring effect:
- decrement `failed`
- increment either `waiting` or `group_waiting`
- increment `waiting_total`
- possibly increment `groups_ready`
- set `last_activity_ms`

## 5.9 Retry failed batch (`retry_failed_batch.lua`)

Same logical behavior as single retry, but in bulk up to the current Lua maximum batch size.

Monitoring and index changes follow the same rules as single retry, aggregated across the successful subset.

## 5.10 Remove job (`remove_job.lua`)

Transactional effect:
- validates expected lane against job state
- rejects removal if job is still active in `{Q}:active`
- removes from the specified lane:
  - `wait`
  - `gwait`
  - `delayed`
  - `failed`
  - `completed`
- deletes `{Q}:job:{job_id}`
- for grouped jobs that still look reserved, decrements `{Q}:g:{gid}:inflight`
- recomputes grouped readiness for `gwait` removals

Index effect:
- removes from the matching `idx:*` key for the lane

Monitoring effect:
- decrements the matching lane counters:
  - `waiting`
  - `group_waiting`
  - `waiting_total`
  - `delayed`
  - `failed`
  - `completed_kept`
- adjusts `groups_ready` when necessary
- sets `last_activity_ms`

## 5.11 Remove jobs batch (`remove_jobs_batch.lua`)

Same logical behavior as single remove, but applied in bulk up to the current Lua maximum batch size.

Counters and group-ready adjustments are aggregated and applied once after processing the batch.

## 5.12 Pause and resume

### Pause (`pause.lua`)

Transactional effect:
- `SET {Q}:paused 1`

Monitoring effect:
- none in `{Q}:stats`

### Resume (`resume.lua`)

Transactional effect:
- `DEL {Q}:paused`

Monitoring effect:
- none in `{Q}:stats`

Paused is intentionally modeled as status-by-key, not a duplicated counter.

---

# 6. Job timing fields and metric semantics

The current job hash stores timing fields that support monitoring and derived metrics.

## 6.1 Stored per-job timestamps

- `created_ms`
  - time when the job hash was created

- `queued_ms`
  - time when the job was initially enqueued

- `first_started_ms`
  - first time the job was ever leased

- `last_started_ms`
  - most recent lease start time

- `completed_ms`
  - terminal success time

- `failed_ms`
  - terminal failure time

- `updated_ms`
  - most recent mutation time

- `last_error_ms`
  - time of last stored error message

## 6.2 Practical interpretations

These fields enable derived metrics such as:

- **initial wait time**
  - `first_started_ms - queued_ms`

- **last wait time before current attempt**
  - `last_started_ms - queued_ms` only if you intentionally treat the full delayed window as waiting

- **processing time for successful jobs**
  - `completed_ms - last_started_ms`

- **processing time for terminally failed jobs**
  - `failed_ms - last_started_ms`

- **end-to-end success latency**
  - `completed_ms - queued_ms`

- **end-to-end terminal failure latency**
  - `failed_ms - queued_ms`

Because retries keep the original `queued_ms`, delayed backoff remains part of total end-to-end waiting/latency.

---

# 7. Exact current queue-level monitoring payload

A manager or monitor can build a queue card from:

- queue name discovered from queue-local `*:stats` keys
- stats from `{Q}:stats`
- pause status from existence of `{Q}:paused`

Example payload:

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
  "last_activity_ms": 1773200000123,
  "last_enqueue_ms": 1773200000000,
  "last_reserve_ms": 1773200000060,
  "last_finish_ms": 1773200000105
}
```


## 7.1 Monitor contract note

The current monitor contract now exposes **two distinct methods** for ready-group reads:

- `groups_ready(queue, offset=0, limit=200) -> list[str]`
  - Returns only group ids currently present in `{Q}:groups:ready`.

- `groups_ready_with_scores(queue, offset=0, limit=200) -> list[GroupReady]`
  - Returns group ids plus their scheduling scores from `{Q}:groups:ready`.

This replaces the older polymorphic style where a single `groups_ready(..., with_scores=...)` method could return different shapes.

### Contract rationale

This split keeps the monitor API explicit and stable across SDKs:

- **simple call** for the common case: just the gids
- **typed detailed call** when the caller also needs the score
- avoids multi-shape return contracts

### Practical mapping to Redis

Both methods read from the same Redis key:

- `{Q}:groups:ready`

They differ only in how the result is shaped for callers.

Example:

```python
gids = monitor.groups_ready("emails")
rows = monitor.groups_ready_with_scores("emails")
```

Conceptually equivalent Go usage:

```go
gids := monitor.GroupsReady("emails", 0, 200)
rows := monitor.GroupsReadyWithScores("emails", 0, 200)
```

---

# 8. Summary of Redis keys

## 8.1 Global keys

No global queue registry key is required by the cluster-safe monitor model.

## 8.2 Per-queue transactional keys

- `{Q}:meta` — anchor key passed to Lua
- `{Q}:wait`
- `{Q}:active`
- `{Q}:delayed`
- `{Q}:failed`
- `{Q}:completed`
- `{Q}:paused`
- `{Q}:groups:ready`
- `{Q}:has_groups`
- `{Q}:lane:rr`
- `{Q}:lease:seq`
- `{Q}:job:{job_id}`
- `{Q}:g:{gid}:wait`
- `{Q}:g:{gid}:inflight`
- `{Q}:g:{gid}:limit`

## 8.3 Per-queue index keys

- `{Q}:idx:wait`
- `{Q}:idx:active`
- `{Q}:idx:delayed`
- `{Q}:idx:failed`
- `{Q}:idx:completed`

## 8.4 Per-queue monitoring keys

- `{Q}:stats`

## 8.5 Child flow helper keys

- `{base}:count`
- `{base}:done`

---

# 9. Final recommendation

The current OmniQ Redis model is now best understood as:

- **transactional lanes and job hashes** for authoritative execution state
- **secondary lane indexes** for management and efficient inspection
- **queue-level stats** for cheap monitoring and UI health views

This gives OmniQ a solid operational foundation for:

- queue monitoring
- manager views
- lane browsing
- retries and removals
- timing-based metrics
- grouped fairness
- Redis/Valkey cluster-safe execution

without introducing a heavy observability-only schema.
