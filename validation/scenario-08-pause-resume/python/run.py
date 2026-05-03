import json
import os
import sys

from omniq.client import OmniqClient

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s08-python")
    first_job = f"{queue}-job-001"
    second_job = f"{queue}-job-002"

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)

    try:
        client.publish(queue=queue, job_id=first_job, payload={"kind": "pause-resume", "n": 1})
        client.publish(queue=queue, job_id=second_job, payload={"kind": "pause-resume", "n": 2})

        paused_before = client.is_paused(queue=queue)
        first = client.reserve(queue=queue)
        if first is None or getattr(first, "status", None) != "JOB":
            raise RuntimeError(f"unexpected first reserve: {first!r}")

        client.pause(queue=queue)
        paused_after_pause = client.is_paused(queue=queue)
        paused_reserve = client.reserve(queue=queue)

        client.resume(queue=queue)
        paused_after_resume = client.is_paused(queue=queue)
        second = client.reserve(queue=queue)
        if second is None or getattr(second, "status", None) != "JOB":
            raise RuntimeError(f"unexpected second reserve: {second!r}")

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "paused_before": paused_before,
                    "paused_after_pause": paused_after_pause,
                    "paused_after_resume": paused_after_resume,
                    "paused_reserve_status": getattr(paused_reserve, "status", None),
                    "first_reserved_job_id": first.job_id,
                    "second_reserved_job_id": second.job_id,
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
