import json
import os
import sys

from omniq.client import OmniqClient


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s02-python")
    job_id = os.environ.get("JOB_ID", f"{queue}-job-001")

    client = OmniqClient(host="omniq-redis", port=6379)

    try:
        client.publish(
            queue=queue,
            job_id=job_id,
            payload={
                "kind": "heartbeat",
                "source": "validation",
                "sdk": "python",
                "value": 2,
            },
            timeout_ms=30_000,
            max_attempts=3,
            backoff_ms=5_000,
        )

        reserve = client.reserve(queue=queue)
        if reserve is None or getattr(reserve, "status", None) != "JOB":
            raise RuntimeError(f"unexpected reserve result: {reserve!r}")

        initial_lock = reserve.lock_until_ms
        new_lock = client.heartbeat(
            queue=queue,
            job_id=reserve.job_id,
            lease_token=reserve.lease_token,
        )

        bad_error = ""
        try:
            client.heartbeat(
                queue=queue,
                job_id=reserve.job_id,
                lease_token="bad-token",
            )
        except Exception as exc:
            bad_error = str(exc)

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "job_id": reserve.job_id,
                    "initial_lock_until_ms": initial_lock,
                    "heartbeat_lock_until_ms": new_lock,
                    "heartbeat_extended": new_lock >= initial_lock,
                    "invalid_token_error": bad_error,
                    "invalid_token_contains_token_mismatch": "TOKEN_MISMATCH" in bad_error,
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
