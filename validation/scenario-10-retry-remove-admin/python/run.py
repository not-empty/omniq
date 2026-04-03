import json
import os
import sys
from dataclasses import asdict

from omniq.client import OmniqClient


def reserve_job(client: OmniqClient, queue: str, now_ms: int):
    res = client.reserve(queue=queue, now_ms_override=now_ms)
    if res is None or getattr(res, "status", None) != "JOB":
        raise RuntimeError(f"unexpected reserve response: {res!r}")
    return res


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s10-python")
    base_now_ms = 1775280000000

    active_job = f"{queue}-active-job-001"
    single_retry_job = f"{queue}-single-retry-job-001"
    batch_retry_job = f"{queue}-batch-retry-job-001"
    waiting_remove_job = f"{queue}-waiting-remove-job-001"
    delayed_remove_job = f"{queue}-delayed-remove-job-001"

    client = OmniqClient(host="omniq-redis", port=6379)

    try:
        client.publish(queue=queue, job_id=active_job, payload={"kind": "admin", "slot": "active"}, max_attempts=3, now_ms_override=base_now_ms + 1)
        client.publish(queue=queue, job_id=single_retry_job, payload={"kind": "admin", "slot": "single-retry"}, max_attempts=1, now_ms_override=base_now_ms + 2)
        client.publish(queue=queue, job_id=batch_retry_job, payload={"kind": "admin", "slot": "batch-retry"}, max_attempts=1, now_ms_override=base_now_ms + 3)
        client.publish(queue=queue, job_id=waiting_remove_job, payload={"kind": "admin", "slot": "waiting-remove"}, max_attempts=3, now_ms_override=base_now_ms + 4)
        client.publish(queue=queue, job_id=delayed_remove_job, payload={"kind": "admin", "slot": "delayed-remove"}, max_attempts=3, due_ms=base_now_ms + 100_000, now_ms_override=base_now_ms + 5)

        active_res = reserve_job(client, queue, base_now_ms + 100)
        single_failed_res = reserve_job(client, queue, base_now_ms + 101)
        batch_failed_res = reserve_job(client, queue, base_now_ms + 102)

        client.ack_fail(
            queue=queue,
            job_id=single_failed_res.job_id,
            lease_token=single_failed_res.lease_token,
            error="single retry setup",
            now_ms_override=base_now_ms + 150,
        )
        client.ack_fail(
            queue=queue,
            job_id=batch_failed_res.job_id,
            lease_token=batch_failed_res.lease_token,
            error="batch retry setup",
            now_ms_override=base_now_ms + 151,
        )

        client.retry_failed(queue=queue, job_id=single_retry_job, now_ms_override=base_now_ms + 200)

        batch_retry_results = client.retry_failed_batch(
            queue=queue,
            job_ids=[batch_retry_job, waiting_remove_job],
            now_ms_override=base_now_ms + 201,
        )

        try:
            client.remove_job(queue=queue, job_id=active_job, lane="wait")
            remove_active_error = "NO_ERROR"
        except Exception as exc:
            remove_active_error = str(exc)

        batch_remove_results = client.remove_jobs_batch(
            queue=queue,
            lane="wait",
            job_ids=[waiting_remove_job, delayed_remove_job],
        )

        delayed_remove_result = client.remove_job(queue=queue, job_id=delayed_remove_job, lane="delayed")

        single_retry_key = f"{{{queue}}}:job:{single_retry_job}"
        raw_state = client.ops.r.hget(single_retry_key, "state")
        raw_attempt = client.ops.r.hget(single_retry_key, "attempt")
        single_retry_state = raw_state.decode("utf-8") if isinstance(raw_state, bytes) else str(raw_state or "")
        single_retry_attempt = int(raw_attempt.decode("utf-8") if isinstance(raw_attempt, bytes) else raw_attempt or 0)

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "single_retry_state": single_retry_state,
                    "single_retry_attempt": single_retry_attempt,
                    "batch_retry_results": [asdict(item) for item in batch_retry_results],
                    "remove_active_error": remove_active_error,
                    "batch_remove_results": [asdict(item) for item in batch_remove_results],
                    "delayed_remove_result": delayed_remove_result,
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
