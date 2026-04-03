import json
import os
import sys
from dataclasses import asdict

from omniq.client import OmniqClient


def reserve_job(client: OmniqClient, queue: str, now_ms: int):
    res = client.reserve(queue=queue, now_ms_override=now_ms)
    if res is None or getattr(res, "status", None) != "JOB":
        raise RuntimeError(f"unexpected reserve response: {res!r}")
    return res


def decode_str(value):
    if isinstance(value, bytes):
        return value.decode("utf-8")
    return str(value or "")


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s21-python")
    base_now_ms = 1775380000000

    wait_job = f"{queue}-wait-job-001"
    grouped_wait_job = f"{queue}-grouped-wait-job-001"
    active_job = f"{queue}-active-job-001"
    delayed_job = f"{queue}-delayed-job-001"
    missing_job = f"{queue}-missing-job-001"

    client = OmniqClient(host="omniq-redis", port=6379)

    try:
        client.publish(queue=queue, job_id=active_job, payload={"kind": "batch-remove-errors", "slot": "active"}, max_attempts=3, now_ms_override=base_now_ms + 1)

        active_res = reserve_job(client, queue, base_now_ms + 100)
        if active_res.job_id != active_job:
            raise RuntimeError(f"expected active job {active_job}, got {active_res.job_id}")

        client.publish(queue=queue, job_id=wait_job, payload={"kind": "batch-remove-errors", "slot": "wait"}, max_attempts=3, now_ms_override=base_now_ms + 2)
        client.publish(queue=queue, job_id=grouped_wait_job, payload={"kind": "batch-remove-errors", "slot": "gwait"}, max_attempts=3, gid="alpha", group_limit=1, now_ms_override=base_now_ms + 3)
        client.publish(queue=queue, job_id=delayed_job, payload={"kind": "batch-remove-errors", "slot": "delayed"}, max_attempts=3, due_ms=base_now_ms + 100_000, now_ms_override=base_now_ms + 4)

        batch_remove_results = client.remove_jobs_batch(
            queue=queue,
            lane="wait",
            job_ids=[wait_job, missing_job, grouped_wait_job, active_job, delayed_job],
        )

        r = client.ops.r
        stats_key = f"{{{queue}}}:stats"

        stats = {
            "waiting": int(decode_str(r.hget(stats_key, "waiting")) or 0),
            "group_waiting": int(decode_str(r.hget(stats_key, "group_waiting")) or 0),
            "waiting_total": int(decode_str(r.hget(stats_key, "waiting_total")) or 0),
            "active": int(decode_str(r.hget(stats_key, "active")) or 0),
            "delayed": int(decode_str(r.hget(stats_key, "delayed")) or 0),
            "groups_ready": int(decode_str(r.hget(stats_key, "groups_ready")) or 0),
        }

        job_hash_exists = {
            "wait_job": int(r.exists(f"{{{queue}}}:job:{wait_job}")),
            "grouped_wait_job": int(r.exists(f"{{{queue}}}:job:{grouped_wait_job}")),
            "active_job": int(r.exists(f"{{{queue}}}:job:{active_job}")),
            "delayed_job": int(r.exists(f"{{{queue}}}:job:{delayed_job}")),
        }

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "batch_remove_results": [asdict(item) for item in batch_remove_results],
                    "job_hash_exists": job_hash_exists,
                    "stats": stats,
                    "wait_len": int(r.llen(f"{{{queue}}}:wait")),
                    "idx_wait": int(r.zcard(f"{{{queue}}}:idx:wait")),
                    "group_wait_len": int(r.llen(f"{{{queue}}}:g:alpha:wait")),
                    "groups_ready": int(r.zcard(f"{{{queue}}}:groups:ready")),
                },
                indent=2,
                sort_keys=True,
            )
        )
        return 0
    finally:
        client.close()


if __name__ == "__main__":
    sys.exit(main())
