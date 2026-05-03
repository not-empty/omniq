import json
import os
import signal
import subprocess
import sys
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


def child_main() -> int:
    queue = os.environ["QUEUE"]
    marker_started = os.environ["MARKER_STARTED"]
    marker_done = os.environ["MARKER_DONE"]

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)
    marker = new_seed()

    def handler(ctx) -> None:
        marker.set(marker_started, "1")
        time.sleep(1.5)
        marker.set(marker_done, "1")

    try:
        client.consume(
            queue=queue,
            handler=handler,
            poll_interval_s=0.02,
            promote_interval_s=10.0,
            reap_interval_s=10.0,
            drain=False,
        )
        return 0
    finally:
        try:
            client.close()
        except Exception:
            pass
        marker.close()


def parent_main() -> int:
    queue = os.environ.get("QUEUE", "validation-s27-python")
    base_now_ms = 1775440000000
    first_job = f"{queue}-job-001"
    second_job = f"{queue}-job-002"
    marker_started = f"{{{queue}}}:marker:started"
    marker_done = f"{{{queue}}}:marker:done"

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)
    inspect = new_seed()

    try:
        client.publish(queue=queue, job_id=first_job, payload={"kind": "drain-false", "slot": 1}, now_ms_override=base_now_ms + 1)
        client.publish(queue=queue, job_id=second_job, payload={"kind": "drain-false", "slot": 2}, now_ms_override=base_now_ms + 2)

        env = os.environ.copy()
        env["QUEUE"] = queue
        env["MARKER_STARTED"] = marker_started
        env["MARKER_DONE"] = marker_done

        child = subprocess.Popen([sys.executable, __file__, "child"], env=env)
        try:
            deadline = time.time() + 5.0
            while time.time() < deadline:
                if inspect.get(marker_started) == "1":
                    break
                time.sleep(0.05)

            os.kill(child.pid, signal.SIGINT)
            try:
                exit_code = child.wait(timeout=5.0)
            except subprocess.TimeoutExpired:
                child.kill()
                exit_code = child.wait(timeout=2.0)
        finally:
            if child.poll() is None:
                child.kill()

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
                    "child_exit_code": exit_code,
                    "handler_started": inspect.get(marker_started) == "1",
                    "handler_done": inspect.get(marker_done) == "1",
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
        inspect.close()


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "child":
        sys.exit(child_main())
    sys.exit(parent_main())
