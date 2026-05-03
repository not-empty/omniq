import json
import os
import sys

import redis

from omniq import OmniqClient

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def new_seed() -> redis.Redis:
    if REDIS_MODE == "cluster":
        return redis.RedisCluster(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    return redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)


def reserve_job(client: OmniqClient, queue: str, now_ms: int):
    res = client.reserve(queue=queue, now_ms_override=now_ms)
    if res is None or getattr(res, "status", None) != "JOB":
        raise RuntimeError(f"unexpected reserve response: {res!r}")
    return res


def decode_str(value):
    if isinstance(value, bytes):
        return value.decode("utf-8")
    return str(value or "")


def script_flush(seed) -> None:
    seed.execute_command("SCRIPT", "FLUSH")


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s25-python")
    base_now_ms = 1775420000000

    publish_job = f"{queue}-job-001"
    delayed_job = f"{queue}-delayed-001"

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)
    seed = new_seed()

    try:
        script_flush(seed)
        published_job_id = client.publish(
            queue=queue,
            job_id=publish_job,
            payload={"kind": "noscript-recovery", "slot": "publish"},
            now_ms_override=base_now_ms + 1,
        )

        script_flush(seed)
        reserved = reserve_job(client, queue, base_now_ms + 100)

        script_flush(seed)
        heartbeat_lock_until_ms = client.heartbeat(
            queue=queue,
            job_id=reserved.job_id,
            lease_token=reserved.lease_token,
            now_ms_override=base_now_ms + 110,
        )

        script_flush(seed)
        client.ack_success(
            queue=queue,
            job_id=reserved.job_id,
            lease_token=reserved.lease_token,
            now_ms_override=base_now_ms + 120,
        )

        script_flush(seed)
        delayed_job_id = client.publish(
            queue=queue,
            job_id=delayed_job,
            payload={"kind": "noscript-recovery", "slot": "delayed"},
            due_ms=base_now_ms + 500,
            now_ms_override=base_now_ms + 2,
        )

        script_flush(seed)
        promoted_count = client.promote_delayed(
            queue=queue,
            max_promote=10,
            now_ms_override=base_now_ms + 600,
        )

        completed_state = decode_str(client.ops.r.hget(f"{{{queue}}}:job:{publish_job}", "state"))
        promoted_state = decode_str(client.ops.r.hget(f"{{{queue}}}:job:{delayed_job}", "state"))

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "published_job_id": published_job_id,
                    "reserved_job_id": reserved.job_id,
                    "heartbeat_lock_until_ms": heartbeat_lock_until_ms,
                    "completed_state": completed_state,
                    "delayed_job_id": delayed_job_id,
                    "promoted_count": promoted_count,
                    "promoted_state": promoted_state,
                },
                indent=2,
                sort_keys=True,
            )
        )
        return 0
    finally:
        client.close()
        try:
            seed.close()
        except Exception:
            pass


if __name__ == "__main__":
    sys.exit(main())
