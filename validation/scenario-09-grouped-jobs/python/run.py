import json
import os
import sys

from omniq.client import OmniqClient

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379
REDIS_MODE = os.environ.get("REDIS_MODE", "standalone")


def reserve_job(client: OmniqClient, queue: str, now_ms: int):
    res = client.reserve(queue=queue, now_ms_override=now_ms)
    if res is None or getattr(res, "status", None) != "JOB":
        raise RuntimeError(f"unexpected reserve response: {res!r}")
    return res


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s09-python")
    base_now_ms = 1775270000000
    alpha_first = f"{queue}-alpha-job-001"
    alpha_second = f"{queue}-alpha-job-002"
    beta_first = f"{queue}-beta-job-001"
    ungrouped = f"{queue}-ungrouped-job-001"

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)

    try:
        client.publish(
            queue=queue,
            job_id=alpha_first,
            payload={"kind": "grouped-jobs", "slot": "alpha-1", "sdk": "python"},
            gid="alpha",
            group_limit=1,
            now_ms_override=base_now_ms + 1,
        )
        client.publish(
            queue=queue,
            job_id=alpha_second,
            payload={"kind": "grouped-jobs", "slot": "alpha-2", "sdk": "python"},
            gid="alpha",
            group_limit=5,
            now_ms_override=base_now_ms + 2,
        )
        client.publish(
            queue=queue,
            job_id=beta_first,
            payload={"kind": "grouped-jobs", "slot": "beta-1", "sdk": "python"},
            gid="beta",
            group_limit=1,
            now_ms_override=base_now_ms + 3,
        )
        client.publish(
            queue=queue,
            job_id=ungrouped,
            payload={"kind": "grouped-jobs", "slot": "ungrouped-1", "sdk": "python"},
            now_ms_override=base_now_ms + 4,
        )

        first = reserve_job(client, queue, base_now_ms + 100)
        second = reserve_job(client, queue, base_now_ms + 101)
        third = reserve_job(client, queue, base_now_ms + 102)
        fourth = client.reserve(queue=queue, now_ms_override=base_now_ms + 103)

        client.ack_success(
            queue=queue,
            job_id=first.job_id,
            lease_token=first.lease_token,
            now_ms_override=base_now_ms + 200,
        )
        fifth = reserve_job(client, queue, base_now_ms + 201)

        group_limit_alpha = client.ops.r.get(f"{{{queue}}}:g:alpha:limit")
        if isinstance(group_limit_alpha, bytes):
            group_limit_alpha = group_limit_alpha.decode("utf-8")

        fourth_status = "EMPTY" if fourth is None else getattr(fourth, "status", None)

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "group_limit_alpha": group_limit_alpha,
                    "reserve_order": [
                        {"job_id": first.job_id, "gid": first.gid},
                        {"job_id": second.job_id, "gid": second.gid},
                        {"job_id": third.job_id, "gid": third.gid},
                    ],
                    "fourth_reserve_status": fourth_status,
                    "fifth_reserve_job_id": fifth.job_id,
                    "fifth_reserve_gid": fifth.gid,
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
