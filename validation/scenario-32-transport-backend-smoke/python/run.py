import json
import os
import sys

from omniq import OmniqClient, QueueMonitor

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s32-python")
    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)
    monitor = QueueMonitor(client)

    try:
        client.publish(
            queue=queue,
            job_id=f"{queue}-job-001",
            payload={"kind": "transport-backend-smoke", "backend": REDIS_MODE, "sdk": "python"},
        )
        reserved = client.reserve(queue=queue)
        if reserved is None or getattr(reserved, "status", None) != "JOB":
            raise RuntimeError(f"unexpected reserve response: {reserved!r}")

        queues_found = sorted([q for q in monitor.scan_queues() if q == queue])
        if queues_found != [queue]:
            raise RuntimeError(f"unexpected discovered queues: {queues_found!r}")

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "backend": REDIS_MODE,
                    "queue": queue,
                    "reserve_status": reserved.status,
                    "reserved_job_id": reserved.job_id,
                    "queues_found": queues_found,
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
