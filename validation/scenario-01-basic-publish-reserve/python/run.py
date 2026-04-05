import json
import os
import sys

from omniq.client import OmniqClient


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-basic-python")
    job_id = os.environ.get("JOB_ID", f"{queue}-job-001")
    payload = {
        "kind": "basic-reserve",
        "source": "validation",
        "sdk": "python",
        "value": 1,
    }

    client = OmniqClient(host="omniq-redis", port=6379)
    invalid_publish_rejected = False

    try:
        try:
            client.publish(queue=queue, payload="raw-string")
        except Exception:
            invalid_publish_rejected = True

        published_job_id = client.publish(
            queue=queue,
            job_id=job_id,
            payload=payload,
            timeout_ms=30_000,
            max_attempts=3,
            backoff_ms=5_000,
        )

        reserve = client.reserve(queue=queue)

        result = {
            "sdk": "python",
            "queue": queue,
            "invalid_publish_rejected": invalid_publish_rejected,
            "job_id": published_job_id,
            "reserve": None
            if reserve is None
            else {
                "status": getattr(reserve, "status", None),
                "job_id": getattr(reserve, "job_id", None),
                "payload": getattr(reserve, "payload", None),
                "attempt": getattr(reserve, "attempt", None),
                "gid": getattr(reserve, "gid", None),
                "lease_token_present": bool(getattr(reserve, "lease_token", "")),
            },
        }
        print(json.dumps(result, indent=2, sort_keys=True))
        return 0
    finally:
        client.close()


if __name__ == "__main__":
    sys.exit(main())
