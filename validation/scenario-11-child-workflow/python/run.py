import json
import os
import sys

from omniq.client import OmniqClient
from omniq.helper import childs_anchor


def main() -> int:
    key = os.environ.get("KEY", "validation-s11-python")
    client = OmniqClient(host="omniq-redis", port=6379)

    try:
        client.childs_init(key=key, expected=3)

        ack_sequence = [
            client.child_ack(key=key, child_id="a"),
            client.child_ack(key=key, child_id="a"),
            client.child_ack(key=key, child_id="b"),
            client.child_ack(key=key, child_id="c"),
        ]

        base = childs_anchor(key)[:-5]
        count_exists_after = client.ops.r.exists(base + ":count")
        done_exists_after = client.ops.r.exists(base + ":done")

        print(
            json.dumps(
                {
                    "sdk": "python",
                    "key": key,
                    "ack_sequence": ack_sequence,
                    "count_exists_after": int(count_exists_after),
                    "done_exists_after": int(done_exists_after),
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
