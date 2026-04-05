import json
import os
import sys

from omniq.client import OmniqClient


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s07-python")
    retry_job_id = os.environ.get("RETRY_JOB_ID", f"{queue}-retry-job-001")
    fail_job_id = os.environ.get("FAIL_JOB_ID", f"{queue}-fail-job-001")
    base_now_ms = 1_775_260_000_000
    reap_now_ms = base_now_ms + 31_000

    client = OmniqClient(host="omniq-redis", port=6379)

    try:
        client.publish(
            queue=queue,
            job_id=retry_job_id,
            payload={"kind": "reap-expired", "mode": "retry", "sdk": "python"},
            timeout_ms=30_000,
            max_attempts=3,
            backoff_ms=5_000,
            now_ms_override=base_now_ms,
        )
        client.publish(
            queue=queue,
            job_id=fail_job_id,
            payload={"kind": "reap-expired", "mode": "terminal", "sdk": "python"},
            timeout_ms=30_000,
            max_attempts=1,
            backoff_ms=5_000,
            now_ms_override=base_now_ms,
        )

        r1 = client.reserve(queue=queue, now_ms_override=base_now_ms)
        r2 = client.reserve(queue=queue, now_ms_override=base_now_ms)
        if r1 is None or r2 is None:
            raise RuntimeError("expected two reserved jobs")

        reaped = client.reap_expired(
            queue=queue,
            max_reap=1000,
            now_ms_override=reap_now_ms,
        )

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "reaped_count": reaped,
                    "retryable_job_id": retry_job_id,
                    "terminal_job_id": fail_job_id,
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
