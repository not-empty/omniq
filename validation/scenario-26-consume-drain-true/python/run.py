import json
import os
import signal
import sys
import threading
import time

import redis

from omniq import OmniqClient

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def new_seed() -> redis.Redis:
    if REDIS_MODE == "cluster":
        return redis.RedisCluster(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    return redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s26-python")
    base_now_ms = 1775430000000
    first_job = f"{queue}-job-001"
    second_job = f"{queue}-job-002"

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)
    inspect = new_seed()

    handled_job_ids: list[str] = []
    started = threading.Event()

    def send_sigint_when_started() -> None:
        started.wait(timeout=5.0)
        if started.is_set():
            time.sleep(0.1)
            os.kill(os.getpid(), signal.SIGINT)

    def handler(ctx) -> None:
        handled_job_ids.append(ctx.job_id)
        if ctx.job_id == first_job:
            started.set()
        time.sleep(0.75)

    try:
        client.publish(queue=queue, job_id=first_job, payload={"kind": "drain-true", "slot": 1}, now_ms_override=base_now_ms + 1)
        client.publish(queue=queue, job_id=second_job, payload={"kind": "drain-true", "slot": 2}, now_ms_override=base_now_ms + 2)

        t = threading.Thread(target=send_sigint_when_started, daemon=True)
        t.start()

        client.consume(
            queue=queue,
            handler=handler,
            poll_interval_s=0.02,
            promote_interval_s=10.0,
            reap_interval_s=10.0,
            drain=True,
        )

        stats_key = f"{{{queue}}}:stats"
        stats = {
            "waiting": int(inspect.hget(stats_key, "waiting") or 0),
            "waiting_total": int(inspect.hget(stats_key, "waiting_total") or 0),
            "active": int(inspect.hget(stats_key, "active") or 0),
            "completed_kept": int(inspect.hget(stats_key, "completed_kept") or 0),
        }

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "handled_job_ids": handled_job_ids,
                    "first_job_state": inspect.hget(f"{{{queue}}}:job:{first_job}", "state") or "",
                    "second_job_state": inspect.hget(f"{{{queue}}}:job:{second_job}", "state") or "",
                    "stats": stats,
                },
                indent=2,
                sort_keys=True,
            )
        )
        return 0
    finally:
        try:
            client.close()
        except Exception:
            pass
        try:
            inspect.close()
        except Exception:
            pass


if __name__ == "__main__":
    sys.exit(main())
