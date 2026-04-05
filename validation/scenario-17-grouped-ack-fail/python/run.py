import json
import os
import sys

from omniq import OmniqClient


def reserve_job(client: OmniqClient, queue: str, now_ms: int):
    res = client.reserve(queue=queue, now_ms_override=now_ms)
    if res is None or getattr(res, "status", None) != "JOB":
        raise RuntimeError(f"unexpected reserve response: {res!r}")
    return res


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s17-python")
    base_now_ms = 1775340000000
    client = OmniqClient(host="omniq-redis", port=6379)

    try:
        client.publish(queue=queue, job_id=f"{queue}-alpha-job-001", payload={"kind": "grouped-ack-fail", "slot": "alpha-1"}, gid="alpha", group_limit=1, max_attempts=3, backoff_ms=5000, now_ms_override=base_now_ms + 1)
        client.publish(queue=queue, job_id=f"{queue}-alpha-job-002", payload={"kind": "grouped-ack-fail", "slot": "alpha-2"}, gid="alpha", group_limit=1, max_attempts=3, backoff_ms=5000, now_ms_override=base_now_ms + 2)
        client.publish(queue=queue, job_id=f"{queue}-beta-job-001", payload={"kind": "grouped-ack-fail", "slot": "beta-1"}, gid="beta", group_limit=1, max_attempts=1, backoff_ms=5000, now_ms_override=base_now_ms + 3)
        client.publish(queue=queue, job_id=f"{queue}-beta-job-002", payload={"kind": "grouped-ack-fail", "slot": "beta-2"}, gid="beta", group_limit=1, max_attempts=1, backoff_ms=5000, now_ms_override=base_now_ms + 4)

        alpha_first = reserve_job(client, queue, base_now_ms + 100)
        beta_first = reserve_job(client, queue, base_now_ms + 101)

        alpha_fail = client.ack_fail(queue=queue, job_id=alpha_first.job_id, lease_token=alpha_first.lease_token, error="retryable grouped fail", now_ms_override=base_now_ms + 150)
        beta_fail = client.ack_fail(queue=queue, job_id=beta_first.job_id, lease_token=beta_first.lease_token, error="terminal grouped fail", now_ms_override=base_now_ms + 151)

        base = f"{{{queue}}}"
        alpha_inflight_after_fail = int(client.ops.r.get(f"{base}:g:alpha:inflight") or 0)
        beta_inflight_after_fail = int(client.ops.r.get(f"{base}:g:beta:inflight") or 0)
        alpha_ready_after_fail = client.ops.r.zscore(f"{base}:groups:ready", "alpha") is not None
        beta_ready_after_fail = client.ops.r.zscore(f"{base}:groups:ready", "beta") is not None

        next_one = reserve_job(client, queue, base_now_ms + 152)
        next_two = reserve_job(client, queue, base_now_ms + 153)

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "alpha_fail_status": alpha_fail[0],
                    "beta_fail_status": beta_fail[0],
                    "alpha_inflight_after_fail": alpha_inflight_after_fail,
                    "beta_inflight_after_fail": beta_inflight_after_fail,
                    "alpha_ready_after_fail": alpha_ready_after_fail,
                    "beta_ready_after_fail": beta_ready_after_fail,
                    "next_job_ids": [next_one.job_id, next_two.job_id],
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
