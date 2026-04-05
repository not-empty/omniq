import json
import os
import sys

from omniq.client import OmniqClient


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s06-python")
    job_id = os.environ.get("JOB_ID", f"{queue}-job-001")
    base_now_ms = 1_775_250_000_000
    due_ms = base_now_ms + 5_000

    client = OmniqClient(host="omniq-redis", port=6379)

    try:
        client.publish(
            queue=queue,
            job_id=job_id,
            payload={
                "kind": "promote-delayed",
                "source": "validation",
                "sdk": "python",
                "value": 6,
            },
            timeout_ms=30_000,
            max_attempts=3,
            backoff_ms=5_000,
            due_ms=due_ms,
            now_ms_override=base_now_ms,
        )

        promoted = client.promote_delayed(
            queue=queue,
            max_promote=1000,
            now_ms_override=due_ms,
        )

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "job_id": job_id,
                    "scheduled_due_ms": due_ms,
                    "promoted_count": promoted,
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
