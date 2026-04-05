import json
import os
import sys
from dataclasses import asdict

from omniq import OmniqClient, QueueMonitor


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s23-python")
    base_now_ms = 1775400000000
    client = OmniqClient(host="omniq-redis", port=6379)
    monitor = QueueMonitor(client)

    gids = ["alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta"]

    try:
        for idx, gid in enumerate(gids, start=1):
            client.publish(
                queue=queue,
                job_id=f"{queue}-{gid}-job-001",
                payload={"kind": "group-pagination", "gid": gid, "slot": 1},
                gid=gid,
                group_limit=1,
                now_ms_override=base_now_ms + idx,
            )

        page_1 = monitor.groups_ready(queue, offset=0, limit=3)
        page_2 = monitor.groups_ready(queue, offset=3, limit=3)
        scored_page_1 = [asdict(x) for x in monitor.groups_ready_with_scores(queue, offset=0, limit=3)]
        scored_page_2 = [asdict(x) for x in monitor.groups_ready_with_scores(queue, offset=3, limit=3)]
        status = [asdict(x) for x in monitor.group_status(queue, ["alpha", "delta", "eta"], default_limit=1)]

        groups_ready_raw = [str(x or "") for x in client.ops.r.zrange(f"{{{queue}}}:groups:ready", 0, -1)]

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "groups_ready_page_1": page_1,
                    "groups_ready_page_2": page_2,
                    "groups_ready_scored_page_1": scored_page_1,
                    "groups_ready_scored_page_2": scored_page_2,
                    "group_status": status,
                    "groups_ready_raw": groups_ready_raw,
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
