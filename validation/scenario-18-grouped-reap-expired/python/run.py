import json
import os
import sys

from omniq import OmniqClient

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def reserve_job(client: OmniqClient, queue: str, now_ms: int):
    res = client.reserve(queue=queue, now_ms_override=now_ms)
    if res is None or getattr(res, "status", None) != "JOB":
        raise RuntimeError(f"unexpected reserve response: {res!r}")
    return res


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s18-python")
    base_now_ms = 1775350000000
    reap_now_ms = base_now_ms + 31_000
    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)

    try:
        client.publish(queue=queue, job_id=f"{queue}-alpha-job-001", payload={"kind": "grouped-reap-expired", "slot": "alpha-1"}, gid="alpha", group_limit=1, max_attempts=3, timeout_ms=30000, backoff_ms=5000, now_ms_override=base_now_ms + 1)
        client.publish(queue=queue, job_id=f"{queue}-alpha-job-002", payload={"kind": "grouped-reap-expired", "slot": "alpha-2"}, gid="alpha", group_limit=1, max_attempts=3, timeout_ms=30000, backoff_ms=5000, now_ms_override=base_now_ms + 2)
        client.publish(queue=queue, job_id=f"{queue}-beta-job-001", payload={"kind": "grouped-reap-expired", "slot": "beta-1"}, gid="beta", group_limit=1, max_attempts=1, timeout_ms=30000, backoff_ms=5000, now_ms_override=base_now_ms + 3)
        client.publish(queue=queue, job_id=f"{queue}-beta-job-002", payload={"kind": "grouped-reap-expired", "slot": "beta-2"}, gid="beta", group_limit=1, max_attempts=1, timeout_ms=30000, backoff_ms=5000, now_ms_override=base_now_ms + 4)

        reserve_job(client, queue, base_now_ms + 100)
        reserve_job(client, queue, base_now_ms + 101)

        reaped_count = client.reap_expired(queue=queue, max_reap=1000, now_ms_override=reap_now_ms)

        base = f"{{{queue}}}"
        alpha_inflight_after_reap = int(client.ops.r.get(f"{base}:g:alpha:inflight") or 0)
        beta_inflight_after_reap = int(client.ops.r.get(f"{base}:g:beta:inflight") or 0)
        alpha_ready_after_reap = client.ops.r.zscore(f"{base}:groups:ready", "alpha") is not None
        beta_ready_after_reap = client.ops.r.zscore(f"{base}:groups:ready", "beta") is not None

        next_one = reserve_job(client, queue, reap_now_ms + 1)
        next_two = reserve_job(client, queue, reap_now_ms + 2)

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "reaped_count": reaped_count,
                    "alpha_inflight_after_reap": alpha_inflight_after_reap,
                    "beta_inflight_after_reap": beta_inflight_after_reap,
                    "alpha_ready_after_reap": alpha_ready_after_reap,
                    "beta_ready_after_reap": beta_ready_after_reap,
                    "next_job_ids": [next_one.job_id, next_two.job_id],
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
