import json
import os
import sys

from omniq import OmniqClient, QueueMonitor

REDIS_HOST = os.environ.get("REDIS_HOST", "omniq-redis")
REDIS_PORT = 6379


def main() -> int:
    queue = os.environ.get("QUEUE", "validation-s29-python")
    invalid_names = [
        "",
        " bad",
        "bad ",
        "bad:name",
        "{manual-tag}",
        "bad/name",
        "bad\\name",
        "bad name",
    ]

    client = OmniqClient(host=REDIS_HOST, port=REDIS_PORT)
    monitor = QueueMonitor(client)

    try:
        valid_job_id = client.publish(
            queue=queue,
            job_id=f"{queue}-job-001",
            payload={"kind": "queue-name-validation", "sdk": "python"},
        )

        invalid_results = []
        for name in invalid_names:
            publish_rejected = False
            stats_rejected = False

            try:
                client.publish(queue=name, payload={"kind": "invalid"})
            except Exception:
                publish_rejected = True

            try:
                monitor.stats(name)
            except Exception:
                stats_rejected = True

            invalid_results.append(
                {
                    "queue": name,
                    "publish_rejected": publish_rejected,
                    "stats_rejected": stats_rejected,
                }
            )

        if not valid_job_id:
            raise RuntimeError("valid queue did not publish a job id")
        if not all(item["publish_rejected"] and item["stats_rejected"] for item in invalid_results):
            raise RuntimeError("invalid queue names were not rejected consistently")

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "queue": queue,
                    "valid_job_id": valid_job_id,
                    "invalid_results": invalid_results,
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
