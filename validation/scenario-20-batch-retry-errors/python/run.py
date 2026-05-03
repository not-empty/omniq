import json
import os
import sys
from dataclasses import asdict

from omniq import OmniqClient

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def reserve_job(client: OmniqClient, queue: str, now_ms: int):
    res = client.reserve(queue=queue, now_ms_override=now_ms)
    if res is None or getattr(res, "status", None) != "JOB":
        raise RuntimeError(f"unexpected reserve response: {res!r}")
    return res


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s20-python")
    base_now_ms = 1775370000000

    failed_job = f"{queue}-failed-job-001"
    waiting_job = f"{queue}-waiting-job-001"
    missing_job = f"{queue}-missing-job-001"

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)

    try:
        client.publish(queue=queue, job_id=failed_job, payload={"kind": "batch-retry-errors", "slot": "failed"}, max_attempts=1, now_ms_override=base_now_ms + 1)
        client.publish(queue=queue, job_id=waiting_job, payload={"kind": "batch-retry-errors", "slot": "waiting"}, max_attempts=3, now_ms_override=base_now_ms + 2)

        failed_res = reserve_job(client, queue, base_now_ms + 100)
        client.ack_fail(
            queue=queue,
            job_id=failed_res.job_id,
            lease_token=failed_res.lease_token,
            error="make failed",
            now_ms_override=base_now_ms + 150,
        )

        batch_retry_results = client.retry_failed_batch(
            queue=queue,
            job_ids=[failed_job, missing_job, waiting_job],
            now_ms_override=base_now_ms + 200,
        )

        retried_job_state = client.ops.r.hget(f"{{{queue}}}:job:{failed_job}", "state")
        if isinstance(retried_job_state, bytes):
            retried_job_state = retried_job_state.decode("utf-8")

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "batch_retry_results": [asdict(item) for item in batch_retry_results],
                    "retried_job_state": retried_job_state,
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
