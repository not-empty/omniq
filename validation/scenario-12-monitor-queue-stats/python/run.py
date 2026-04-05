import json
import os
import sys
from dataclasses import asdict

from omniq import OmniqClient, QueueMonitor


def reserve_job(client: OmniqClient, queue: str, now_ms: int):
    res = client.reserve(queue=queue, now_ms_override=now_ms)
    if res is None or getattr(res, "status", None) != "JOB":
        raise RuntimeError(f"unexpected reserve response: {res!r}")
    return res


def main() -> int:
    prefix = os.environ.get("PREFIX", "validation-s12-python")
    queue_a = f"{prefix}-paused"
    queue_b = f"{prefix}-mixed"
    base_now_ms = 1775290000000

    client = OmniqClient(host="omniq-redis", port=6379)
    monitor = QueueMonitor(client)

    try:
        client.publish(queue=queue_a, job_id=f"{queue_a}-job-001", payload={"kind": "monitor", "queue": "a"}, now_ms_override=base_now_ms + 1)
        client.pause(queue=queue_a)

        completed_job = f"{queue_b}-completed-job-001"
        active_job = f"{queue_b}-active-job-001"
        delayed_job = f"{queue_b}-delayed-job-001"

        client.publish(queue=queue_b, job_id=completed_job, payload={"kind": "monitor", "slot": "completed"}, now_ms_override=base_now_ms + 2)
        client.publish(queue=queue_b, job_id=active_job, payload={"kind": "monitor", "slot": "active"}, now_ms_override=base_now_ms + 3)
        client.publish(queue=queue_b, job_id=delayed_job, payload={"kind": "monitor", "slot": "delayed"}, due_ms=base_now_ms + 100_000, now_ms_override=base_now_ms + 4)

        completed_res = reserve_job(client, queue_b, base_now_ms + 100)
        active_res = reserve_job(client, queue_b, base_now_ms + 101)
        client.ack_success(queue=queue_b, job_id=completed_res.job_id, lease_token=completed_res.lease_token, now_ms_override=base_now_ms + 150)
        _ = active_res

        list_queues = monitor.list_queues()
        queues_found = sorted([q for q in list_queues if q in {queue_a, queue_b}])
        stats_a = asdict(monitor.stats(queue_a))
        stats_b = asdict(monitor.stats(queue_b))
        stats_many = [asdict(x) for x in monitor.stats_many([queue_a, queue_b])]

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queues_found": queues_found,
                    "stats_a": stats_a,
                    "stats_b": stats_b,
                    "stats_many": stats_many,
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
