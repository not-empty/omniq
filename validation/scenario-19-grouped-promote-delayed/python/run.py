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
    queue = os.environ.get("QUEUE", "validation-s19-python")
    base_now_ms = 1775360000000
    due_ms = base_now_ms + 5000
    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)

    try:
        client.publish(queue=queue, job_id=f"{queue}-alpha-job-001", payload={"kind": "grouped-promote-delayed", "slot": "alpha-1"}, gid="alpha", group_limit=1, due_ms=due_ms, now_ms_override=base_now_ms + 1)
        client.publish(queue=queue, job_id=f"{queue}-beta-job-001", payload={"kind": "grouped-promote-delayed", "slot": "beta-1"}, gid="beta", group_limit=1, due_ms=due_ms, now_ms_override=base_now_ms + 2)

        promoted_count = client.promote_delayed(queue=queue, max_promote=1000, now_ms_override=due_ms)

        base = f"{{{queue}}}"
        alpha_ready_after_promote = client.ops.r.zscore(f"{base}:groups:ready", "alpha") is not None
        beta_ready_after_promote = client.ops.r.zscore(f"{base}:groups:ready", "beta") is not None
        stats_raw = client.ops.r.hgetall(f"{base}:stats") or {}
        group_waiting_after_promote = int(stats_raw.get("group_waiting") or 0)

        next_one = reserve_job(client, queue, due_ms + 1)
        next_two = reserve_job(client, queue, due_ms + 2)

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "promoted_count": promoted_count,
                    "alpha_ready_after_promote": alpha_ready_after_promote,
                    "beta_ready_after_promote": beta_ready_after_promote,
                    "group_waiting_after_promote": group_waiting_after_promote,
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
