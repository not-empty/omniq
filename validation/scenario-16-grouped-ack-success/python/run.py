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
    queue = os.environ.get("QUEUE", "validation-s16-python")
    base_now_ms = 1775330000000
    gid = "alpha"
    first_job = f"{queue}-alpha-job-001"
    second_job = f"{queue}-alpha-job-002"

    client = OmniqClient(host="omniq-redis", port=6379)

    try:
        client.publish(
            queue=queue,
            job_id=first_job,
            payload={"kind": "grouped-ack-success", "slot": "first"},
            gid=gid,
            group_limit=1,
            now_ms_override=base_now_ms + 1,
        )
        client.publish(
            queue=queue,
            job_id=second_job,
            payload={"kind": "grouped-ack-success", "slot": "second"},
            gid=gid,
            group_limit=1,
            now_ms_override=base_now_ms + 2,
        )

        first = reserve_job(client, queue, base_now_ms + 100)
        client.ack_success(
            queue=queue,
            job_id=first.job_id,
            lease_token=first.lease_token,
            now_ms_override=base_now_ms + 150,
        )

        base = f"{{{queue}}}"
        group_ready_after_ack = client.ops.r.zscore(f"{base}:groups:ready", gid) is not None
        inflight_raw = client.ops.r.get(f"{base}:g:{gid}:inflight")
        group_inflight_after_ack = int(inflight_raw or 0)

        second = reserve_job(client, queue, base_now_ms + 151)

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "first_job_id": first.job_id,
                    "second_job_id": second.job_id,
                    "group_ready_after_ack": group_ready_after_ack,
                    "group_inflight_after_ack": group_inflight_after_ack,
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
