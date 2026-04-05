import json
import os
import sys
from dataclasses import asdict

import redis

from omniq import OmniqClient, QueueMonitor


def main() -> int:
    prefix = os.environ.get("PREFIX", "validation-s24-python")
    queue_empty = f"{prefix}-empty"
    queue_partial = f"{prefix}-partial"
    queue_paused = f"{prefix}-paused"

    client = OmniqClient(host="omniq-redis", port=6379)
    monitor = QueueMonitor(client)
    seed = redis.Redis(host="omniq-redis", port=6379, decode_responses=True)

    try:
        seed.sadd("omniq:queues", f"{{{queue_empty}}}", f"{{{queue_partial}}}", f"{{{queue_paused}}}")
        seed.hset(
            f"{{{queue_partial}}}:stats",
            mapping={
                "waiting": "2",
                "group_waiting": "1",
                "active": "3",
                "last_activity_ms": "1775410000001",
            },
        )
        seed.set(f"{{{queue_paused}}}:paused", "1")

        queues_found = sorted([q for q in monitor.list_queues() if q in {queue_empty, queue_partial, queue_paused}])
        stats_empty = asdict(monitor.stats(queue_empty))
        stats_partial = asdict(monitor.stats(queue_partial))
        stats_paused = asdict(monitor.stats(queue_paused))
        stats_many = [asdict(x) for x in monitor.stats_many([queue_empty, queue_partial, queue_paused])]

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queues_found": queues_found,
                    "stats_empty": stats_empty,
                    "stats_partial": stats_partial,
                    "stats_paused": stats_paused,
                    "stats_many": stats_many,
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
