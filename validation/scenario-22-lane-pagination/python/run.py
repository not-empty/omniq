import json
import os
import sys
from dataclasses import asdict

from omniq import OmniqClient, QueueMonitor


def job_ids(rows):
    return [row["job_id"] if isinstance(row, dict) else row.job_id for row in rows]


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s22-python")
    base_now_ms = 1775390000000

    wait_jobs = [f"{queue}-wait-{i:03d}" for i in range(1, 6)]
    delayed_jobs = [f"{queue}-delayed-{i:03d}" for i in range(1, 6)]

    client = OmniqClient(host="omniq-redis", port=6379)
    monitor = QueueMonitor(client)

    try:
        for idx, job_id in enumerate(wait_jobs, start=1):
            client.publish(
                queue=queue,
                job_id=job_id,
                payload={"kind": "lane-pagination", "lane": "wait", "order": idx},
                now_ms_override=base_now_ms + idx,
            )

        for idx, job_id in enumerate(delayed_jobs, start=1):
            client.publish(
                queue=queue,
                job_id=job_id,
                payload={"kind": "lane-pagination", "lane": "delayed", "order": idx},
                due_ms=base_now_ms + 100_000 + idx,
                now_ms_override=base_now_ms + 100 + idx,
            )

        wait_forward_pages = [
            [asdict(x) for x in monitor.lane_page(queue, "wait", offset=0, limit=2, reverse=False)],
            [asdict(x) for x in monitor.lane_page(queue, "wait", offset=2, limit=2, reverse=False)],
            [asdict(x) for x in monitor.lane_page(queue, "wait", offset=4, limit=2, reverse=False)],
        ]
        wait_reverse_pages = [
            [asdict(x) for x in monitor.lane_page(queue, "wait", offset=0, limit=2, reverse=True)],
            [asdict(x) for x in monitor.lane_page(queue, "wait", offset=2, limit=2, reverse=True)],
            [asdict(x) for x in monitor.lane_page(queue, "wait", offset=4, limit=2, reverse=True)],
        ]
        delayed_forward_pages = [
            [asdict(x) for x in monitor.lane_page(queue, "delayed", offset=0, limit=2, reverse=False)],
            [asdict(x) for x in monitor.lane_page(queue, "delayed", offset=2, limit=2, reverse=False)],
            [asdict(x) for x in monitor.lane_page(queue, "delayed", offset=4, limit=2, reverse=False)],
        ]
        delayed_reverse_pages = [
            [asdict(x) for x in monitor.lane_page(queue, "delayed", offset=0, limit=2, reverse=True)],
            [asdict(x) for x in monitor.lane_page(queue, "delayed", offset=2, limit=2, reverse=True)],
            [asdict(x) for x in monitor.lane_page(queue, "delayed", offset=4, limit=2, reverse=True)],
        ]

        stats = asdict(monitor.stats(queue))
        idx_wait_raw = [str(x or "") for x in client.ops.r.zrange(f"{{{queue}}}:idx:wait", 0, -1)]
        idx_delayed_raw = [str(x or "") for x in client.ops.r.zrange(f"{{{queue}}}:idx:delayed", 0, -1)]

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "stats": stats,
                    "wait_forward_pages": [job_ids(page) for page in wait_forward_pages],
                    "wait_reverse_pages": [job_ids(page) for page in wait_reverse_pages],
                    "delayed_forward_pages": [job_ids(page) for page in delayed_forward_pages],
                    "delayed_reverse_pages": [job_ids(page) for page in delayed_reverse_pages],
                    "idx_wait_raw": idx_wait_raw,
                    "idx_delayed_raw": idx_delayed_raw,
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
