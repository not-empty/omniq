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
    queue = os.environ.get("QUEUE", "validation-s14-python")
    base_now_ms = 1775310000000

    wait_keep = f"{queue}-wait-keep-001"
    wait_missing = f"{queue}-wait-missing-001"
    active_job = f"{queue}-active-001"
    delayed_job = f"{queue}-delayed-001"
    completed_job = f"{queue}-completed-001"
    failed_job = f"{queue}-failed-001"

    client = OmniqClient(host="omniq-redis", port=6379)
    monitor = QueueMonitor(client)

    try:
        client.publish(queue=queue, job_id=completed_job, payload={"kind": "monitor-lanes", "slot": "completed"}, now_ms_override=base_now_ms + 1)
        client.publish(queue=queue, job_id=active_job, payload={"kind": "monitor-lanes", "slot": "active"}, now_ms_override=base_now_ms + 2)
        client.publish(queue=queue, job_id=failed_job, payload={"kind": "monitor-lanes", "slot": "failed"}, max_attempts=1, now_ms_override=base_now_ms + 3)
        client.publish(queue=queue, job_id=delayed_job, payload={"kind": "monitor-lanes", "slot": "delayed"}, due_ms=base_now_ms + 100_000, now_ms_override=base_now_ms + 4)
        client.publish(queue=queue, job_id=wait_keep, payload={"kind": "monitor-lanes", "slot": "wait-keep"}, now_ms_override=base_now_ms + 5)
        client.publish(queue=queue, job_id=wait_missing, payload={"kind": "monitor-lanes", "slot": "wait-missing"}, now_ms_override=base_now_ms + 6)

        completed_res = reserve_job(client, queue, base_now_ms + 100)
        active_res = reserve_job(client, queue, base_now_ms + 101)
        failed_res = reserve_job(client, queue, base_now_ms + 102)

        client.ack_success(queue=queue, job_id=completed_res.job_id, lease_token=completed_res.lease_token, now_ms_override=base_now_ms + 150)
        client.ack_fail(queue=queue, job_id=failed_res.job_id, lease_token=failed_res.lease_token, error="terminal failure", now_ms_override=base_now_ms + 151)

        missing_job_key = f"{{{queue}}}:job:{wait_missing}"
        client.ops.r.delete(missing_job_key)

        wait_page = [asdict(x) for x in monitor.lane_page(queue, "wait", offset=0, limit=10, reverse=False)]
        wait_page_reverse = [asdict(x) for x in monitor.lane_page(queue, "wait", offset=0, limit=10, reverse=True)]
        find_wait = [asdict(x) for x in monitor.find_jobs(queue, "wait", [wait_keep, wait_missing])]
        get_existing = asdict(monitor.get_job(queue, active_job))
        get_missing = monitor.get_job(queue, wait_missing)
        overview = asdict(monitor.overview(queue, samples_per_lane=10))

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "wait_page": wait_page,
                    "wait_page_reverse": wait_page_reverse,
                    "find_wait": find_wait,
                    "get_existing": get_existing,
                    "get_missing": get_missing,
                    "overview": overview,
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
