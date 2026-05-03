import json
import os
import sys
from dataclasses import asdict

import redis

from omniq import OmniqClient, QueueMonitor

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def new_seed():
    if REDIS_MODE == "cluster":
        return redis.RedisCluster(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    return redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)


def main() -> int:
    prefix = os.environ.get("PREFIX", "validation-s30-python")
    queue_a = f"{prefix}-alpha"
    queue_b = f"{prefix}.beta_2"
    paused_only = f"{prefix}-paused-only"
    invalid_colon_key = f"{prefix}-bad:name:stats"
    invalid_space_key = f"{{{prefix} bad}}:stats"

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)
    monitor = QueueMonitor(client)
    seed = new_seed()

    try:
        seed.hset(f"{{{queue_a}}}:stats", mapping={"waiting": "0"})
        seed.hset(f"{{{queue_b}}}:stats", mapping={"waiting": "1"})
        seed.set(f"{{{paused_only}}}:paused", "1")
        seed.hset(invalid_colon_key, mapping={"waiting": "9"})
        seed.hset(invalid_space_key, mapping={"waiting": "9"})

        queues_found = sorted([q for q in monitor.scan_queues() if q.startswith(prefix)])
        stats_many_auto = [asdict(x)["queue"] for x in monitor.stats_many() if x.queue.startswith(prefix)]
        expected = sorted([queue_a, queue_b])

        if queues_found != expected:
            raise RuntimeError(f"unexpected discovered queues: {queues_found!r}")
        if sorted(stats_many_auto) != expected:
            raise RuntimeError(f"unexpected stats_many() discovery: {stats_many_auto!r}")
        if paused_only in queues_found:
            raise RuntimeError("paused-only queue should not be discovered")
        if any("bad" in q for q in queues_found):
            raise RuntimeError("invalid sparse keys leaked into queue discovery")

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queues_found": queues_found,
                    "stats_many_auto": stats_many_auto,
                    "paused_only_discovered": paused_only in queues_found,
                    "invalid_discovered": any("bad" in q for q in queues_found),
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
