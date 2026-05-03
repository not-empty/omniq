import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

const REDIS_HOST = process.env.REDIS_HOST ?? "omniq-redis";
const REDIS_PORT = 6379;
const REDIS_MODE = process.env.REDIS_MODE ?? "standalone";
import { childsAnchor } from "/workspace/omniq-node/src/helper.ts";

async function main() {
  const key = process.env.KEY ?? "validation-s11-node";

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.childs_init({ key, expected: 3 });

    const ackSequence = [
      await client.child_ack({ key, child_id: "a" }),
      await client.child_ack({ key, child_id: "a" }),
      await client.child_ack({ key, child_id: "b" }),
      await client.child_ack({ key, child_id: "c" }),
    ];

    const base = childsAnchor(key).slice(0, -5);
    const rawRedis = (client as any).ops.r;
    const countExistsAfter = Number(await rawRedis.exists(`${base}:count`));
    const doneExistsAfter = Number(await rawRedis.exists(`${base}:done`));

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          key,
          ack_sequence: ackSequence,
          count_exists_after: countExistsAfter,
          done_exists_after: doneExistsAfter,
        },
        null,
        2
      )
    );
  } finally {
    await client.close();
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
