import json
import os
import sys

from omniq import OmniqClient


def reserve_job(client: OmniqClient, queue: str, now_ms: int):
    res = client.reserve(queue=queue, now_ms_override=now_ms)
    if res is None or getattr(res, "status", None) != "JOB":
        raise RuntimeError(f"unexpected reserve response: {res!r}")
    return res


def capture(fn):
    try:
        fn()
        return "NO_ERROR"
    except Exception as exc:
        return str(exc)


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s15-python")
    base_now_ms = 1775320000000

    job_id = f"{queue}-job-001"
    delayed_job = f"{queue}-delayed-001"

    client = OmniqClient(host="omniq-redis", port=6379)

    try:
        invalid_publish = capture(lambda: client.publish(queue=queue, job_id=f"{queue}-bad-publish", payload="raw-string"))

        client.publish(queue=queue, job_id=job_id, payload={"kind": "error-surface"}, now_ms_override=base_now_ms + 1)
        client.publish(queue=queue, job_id=delayed_job, payload={"kind": "error-surface", "slot": "delayed"}, due_ms=base_now_ms + 100_000, now_ms_override=base_now_ms + 2)

        reserved = reserve_job(client, queue, base_now_ms + 100)

        token_mismatch = capture(
            lambda: client.ack_success(
                queue=queue,
                job_id=reserved.job_id,
                lease_token="bad-token",
                now_ms_override=base_now_ms + 110,
            )
        )

        active_key = f"{{{queue}}}:active"
        client.ops.r.zrem(active_key, reserved.job_id)

        not_active = capture(
            lambda: client.ack_success(
                queue=queue,
                job_id=reserved.job_id,
                lease_token=reserved.lease_token,
                now_ms_override=base_now_ms + 112,
            )
        )

        batch_limit = capture(
            lambda: client.retry_failed_batch(
                queue=queue,
                job_ids=[f"{queue}-x-{i:03d}" for i in range(101)],
                now_ms_override=base_now_ms + 120,
            )
        )

        lane_mismatch = capture(
            lambda: client.remove_job(
                queue=queue,
                job_id=delayed_job,
                lane="wait",
            )
        )

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "token_mismatch": token_mismatch,
                    "not_active": not_active,
                    "batch_limit": batch_limit,
                    "invalid_publish": invalid_publish,
                    "lane_mismatch": lane_mismatch,
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
