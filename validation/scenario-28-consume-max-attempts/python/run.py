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
    queue = os.environ.get("QUEUE", "validation-s28-python")
    job_id = f"{queue}-job-001"
    base_now_ms = 1775440000000

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)
    inspect = new_seed()

    seen: list[dict[str, object]] = []
    sig_sent = False

    def handler(ctx) -> None:
        nonlocal sig_sent
        is_last_attempt = ctx.attempt >= ctx.max_attempts
        seen.append(
            {
                "attempt": ctx.attempt,
                "max_attempts": ctx.max_attempts,
                "is_last_attempt": is_last_attempt,
            }
        )

        if not is_last_attempt:
            raise RuntimeError("Intentional failure before the last attempt")

        if not sig_sent:
            sig_sent = True
            threading.Timer(0.05, lambda: os.kill(os.getpid(), signal.SIGINT)).start()
        time.sleep(0.1)

    try:
        client.publish(
            queue=queue,
            job_id=job_id,
            payload={"kind": "consume-max-attempts", "sdk": "python"},
            max_attempts=3,
            backoff_ms=100,
            timeout_ms=30_000,
            now_ms_override=base_now_ms + 1,
        )

        client.consume(
            queue=queue,
            handler=handler,
            poll_interval_s=0.02,
            promote_interval_s=0.05,
            reap_interval_s=10.0,
            drain=True,
        )

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "job_id": job_id,
                    "seen": seen,
                    "final_state": inspect.hget(f"{{{queue}}}:job:{job_id}", "state") or "",
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
