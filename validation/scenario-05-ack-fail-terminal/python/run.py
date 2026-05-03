import json
import os
import sys

from omniq.client import OmniqClient

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s05-python")
    job_id = os.environ.get("JOB_ID", f"{queue}-job-001")

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)

    try:
        client.publish(
            queue=queue,
            job_id=job_id,
            payload={
                "kind": "ack-fail-terminal",
                "source": "validation",
                "sdk": "python",
                "value": 5,
            },
            timeout_ms=30_000,
            max_attempts=1,
            backoff_ms=5_000,
        )

        reserve = client.reserve(queue=queue)
        if reserve is None or getattr(reserve, "status", None) != "JOB":
            raise RuntimeError(f"unexpected reserve result: {reserve!r}")

        bad_error = ""
        try:
            client.ack_fail(
                queue=queue,
                job_id=reserve.job_id,
                lease_token="bad-token",
                error="boom-terminal",
            )
        except Exception as exc:
            bad_error = str(exc)

        status, due_ms = client.ack_fail(
            queue=queue,
            job_id=reserve.job_id,
            lease_token=reserve.lease_token,
            error="boom-terminal",
        )

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "job_id": reserve.job_id,
                    "ack_fail_status": status,
                    "due_ms": due_ms,
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
