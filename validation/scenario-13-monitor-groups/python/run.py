import json
import os
import sys
from dataclasses import asdict

from omniq import OmniqClient, QueueMonitor

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s13-python")
    base_now_ms = 1775300000000
    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)
    monitor = QueueMonitor(client)

    try:
        client.publish(queue=queue, job_id=f"{queue}-alpha-job-001", payload={"kind": "monitor-groups", "slot": "alpha-1"}, gid="alpha", group_limit=2, now_ms_override=base_now_ms + 1)
        client.publish(queue=queue, job_id=f"{queue}-alpha-job-002", payload={"kind": "monitor-groups", "slot": "alpha-2"}, gid="alpha", group_limit=2, now_ms_override=base_now_ms + 2)
        client.publish(queue=queue, job_id=f"{queue}-beta-job-001", payload={"kind": "monitor-groups", "slot": "beta-1"}, gid="beta", group_limit=1, now_ms_override=base_now_ms + 3)
        client.publish(queue=queue, job_id=f"{queue}-gamma-job-001", payload={"kind": "monitor-groups", "slot": "gamma-1"}, gid="gamma", group_limit=1, now_ms_override=base_now_ms + 4)
        client.publish(queue=queue, job_id=f"{queue}-delta-job-001", payload={"kind": "monitor-groups", "slot": "delta-1"}, gid="delta", group_limit=1, now_ms_override=base_now_ms + 5)

        first = client.reserve(queue=queue, now_ms_override=base_now_ms + 100)
        if first is None or getattr(first, "status", None) != "JOB":
            raise RuntimeError(f"unexpected reserve response: {first!r}")

        gids = ["alpha", "beta", "gamma", "delta"]
        groups_ready_page = monitor.groups_ready(queue, offset=0, limit=2)
        groups_ready_all = [asdict(x) for x in monitor.groups_ready_with_scores(queue, offset=0, limit=10)]
        group_status = [asdict(x) for x in monitor.group_status(queue, gids, default_limit=1)]

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "groups_ready_page": groups_ready_page,
                    "groups_ready_all": groups_ready_all,
                    "group_status": group_status,
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
