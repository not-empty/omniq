import json
import os
import sys

import redis

from omniq import OmniqClient, QueueMonitor

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def new_seed():
    if REDIS_MODE == "cluster":
        return redis.RedisCluster(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    return redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)


def script_flush(seed) -> None:
    seed.execute_command("SCRIPT", "FLUSH")


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
    queue_prefix = os.environ.get("QUEUE", "validation-s31-python")
    queue_a = f"{queue_prefix}-a"
    queue_b = f"{queue_prefix}-b"
    base_now_ms = 1775450000000

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)
    monitor = QueueMonitor(client)
    seed = new_seed()

    try:
        script_flush(seed)
        client.publish(
            queue=queue_a,
            job_id=f"{queue_a}-job-001",
            payload={"kind": "multi-queue-noscript", "queue": "a"},
            now_ms_override=base_now_ms + 1,
        )

        script_flush(seed)
        client.publish(
            queue=queue_b,
            job_id=f"{queue_b}-job-001",
            payload={"kind": "multi-queue-noscript", "queue": "b"},
            now_ms_override=base_now_ms + 2,
        )

        script_flush(seed)
        reserved_a = reserve_job(client, queue_a, base_now_ms + 100)

        script_flush(seed)
        client.ack_success(
            queue=queue_a,
            job_id=reserved_a.job_id,
            lease_token=reserved_a.lease_token,
            now_ms_override=base_now_ms + 110,
        )

        script_flush(seed)
        reserved_b = reserve_job(client, queue_b, base_now_ms + 120)

        script_flush(seed)
        heartbeat_b = client.heartbeat(
            queue=queue_b,
            job_id=reserved_b.job_id,
            lease_token=reserved_b.lease_token,
            now_ms_override=base_now_ms + 130,
        )

        script_flush(seed)
        client.ack_success(
            queue=queue_b,
            job_id=reserved_b.job_id,
            lease_token=reserved_b.lease_token,
            now_ms_override=base_now_ms + 140,
        )

        queues_found = sorted([q for q in monitor.scan_queues() if q in {queue_a, queue_b}])
        queue_a_state = decode_str(client.ops.r.hget(f"{{{queue_a}}}:job:{queue_a}-job-001", "state"))
        queue_b_state = decode_str(client.ops.r.hget(f"{{{queue_b}}}:job:{queue_b}-job-001", "state"))

        if queues_found != [queue_a, queue_b]:
            raise RuntimeError(f"unexpected discovered queues: {queues_found!r}")
        if queue_a_state != "completed" or queue_b_state != "completed":
            raise RuntimeError("multi-queue NOSCRIPT flow did not complete both jobs")
        if heartbeat_b <= 0:
            raise RuntimeError("heartbeat did not extend queue B lease")

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queues_found": queues_found,
                    "queue_a_state": queue_a_state,
                    "queue_b_state": queue_b_state,
                    "heartbeat_b": heartbeat_b,
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
