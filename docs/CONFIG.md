# CONFIG.md — OmniQ v1

## Purpose

Defines the configuration contract for queues and jobs in **OmniQ v1**.

OmniQ v1 supports:
- **Ungrouped jobs** by default
- **Optional groups** (FIFO per group + per-group concurrency)
- **Queue pause / resume** (flag-only; does not move jobs)

---

## Queue configuration (defaults)

```yaml
queue: pdf-processing
timeout_ms: 300000
max_attempts: 5
backoff_ms: 30000
completed_keep: 100
default_group_limit: 1
```

| Field | Description |
|---|---|
| queue | Queue name |
| timeout_ms | Lease duration (used to compute `lock_until_ms`) |
| max_attempts | Retry limit (`attempt >= max_attempts` → terminal failed) |
| backoff_ms | Fixed retry delay (ms) |
| completed_keep | Retention size for `{Q}:completed` |
| default_group_limit | Default per-group concurrency if not set |

Notes:
- `completed_keep` is enforced in Lua via trim logic
- Pause / resume has **no static config**; it is a runtime flag

---

## Job enqueue (ungrouped)

```json
{
  "queue": "pdf-processing",
  "payload": {}
}
```

### Optional overrides
```json
{
  "job_id": "01J8Z6PZQF7Y9X8E4G9K1H2ABC",
  "timeout_ms": 120000,
  "max_attempts": 3,
  "backoff_ms": 15000,
  "due_ms": 0
}
```

---

## Job enqueue (grouped — opt-in)

Grouped jobs are enabled by providing a `gid`.

```json
{
  "queue": "pdf-processing",
  "payload": {},
  "gid": "company:acme",
  "group_limit": 2
}
```

Notes:
- `gid` routes jobs to `{Q}:g:{gid}:wait`
- `group_limit` lazily initializes `{Q}:g:{gid}:limit`
- Existing limits are never overridden (first writer wins)

---

## Retry policy

- Fixed backoff (`backoff_ms`)
- Retries are scheduled in `{Q}:delayed`
- `PROMOTE_DELAYED` routes jobs back to the correct lane

---

## Timeout & leases

- Lease-based execution (`lock_until_ms`)
- Active leases tracked in `{Q}:active`

### Heartbeats & stalled recovery

- `RESERVE` sets `lock_until_ms = now + timeout_ms`
- Workers SHOULD heartbeat periodically
- Job is stalled when `now_ms > lock_until_ms`
- `REAP_EXPIRED` reclaims stalled jobs and applies retry / terminal logic

Recommended heartbeat interval:
- `timeout_ms / 2` (derived, not a config field)

---

## Pause / resume (flag-only)

Pause / resume in OmniQ v1 is intentionally simple and safe.

### Behavior
- Pause does NOT move jobs
- Pause does NOT affect running jobs
- Pause only blocks **new reserves**

### Redis key
- `{Q}:paused` (STRING)

### Commands
- `PAUSE` → creates `{Q}:paused`
- `RESUME` → deletes `{Q}:paused`

### Worker semantics
- `RESERVE` returns `PAUSED` if flag exists
- Workers SHOULD back off before retrying reserve

### Race semantics
- One job may start while pause is being set
- This is expected and correct behavior

This design avoids stuck-job scenarios caused by moving jobs between lists.

---

## Explicit non-goals (v1)

- Priority queues
- DAG / workflow orchestration
- Cron scheduling (beyond delayed retries)
- Per-job paused state
- UI-style paused lists
