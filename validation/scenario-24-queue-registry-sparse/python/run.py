import json
import os
import sys
from dataclasses import asdict

import redis

from omniq import OmniqClient, QueueMonitor

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def new_seed() -> redis.Redis:
    if REDIS_MODE == "cluster":
        return redis.RedisCluster(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    return redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def main() -> int:
    prefix = os.environ.get("PREFIX", "validation-s24-python")
    queue_empty = f"{prefix}-empty"
    queue_partial = f"{prefix}-partial"
    queue_paused = f"{prefix}-paused"

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)
    monitor = QueueMonitor(client)
    seed = new_seed()

    try:
        seed.hset(f"{{{queue_empty}}}:stats", mapping={"waiting": "0"})
        seed.hset(
            f"{{{queue_partial}}}:stats",
            mapping={
                "waiting": "2",
                "group_waiting": "1",
                "active": "3",
                "last_activity_ms": "1775410000001",
            },
        )
        seed.hset(f"{{{queue_paused}}}:stats", mapping={"waiting": "0"})
        seed.set(f"{{{queue_paused}}}:paused", "1")

        queues_found = sorted([q for q in monitor.scan_queues() if q in {queue_empty, queue_partial, queue_paused}])
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
