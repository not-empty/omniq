import Redis from "/workspace/omniq-node/node_modules/ioredis/built/index.js";
import { OmniqClient, QueueMonitor } from "/workspace/omniq-node/src/index.ts";

const REDIS_HOST = process.env.REDIS_HOST ?? "omniq-redis";
const REDIS_PORT = 6379;
const REDIS_MODE = process.env.REDIS_MODE ?? "standalone";

function newRawRedis() {
  if (REDIS_MODE === "cluster") {
    return new (Redis as any).Cluster([{ host: REDIS_HOST, port: REDIS_PORT }]);
  }
  return new Redis({ host: REDIS_HOST, port: REDIS_PORT });
}

async function main() {
  const prefix = process.env.PREFIX ?? "validation-s24-node";
  const queueEmpty = `${prefix}-empty`;
  const queuePartial = `${prefix}-partial`;
  const queuePaused = `${prefix}-paused`;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const monitor = new QueueMonitor(client);
  const seed = newRawRedis();

  try {
    await seed.hset(`{${queueEmpty}}:stats`, { waiting: "0" });
    await seed.hset(`{${queuePartial}}:stats`, {
      waiting: "2",
      group_waiting: "1",
      active: "3",
      last_activity_ms: "1775410000001",
    });
    await seed.set(`{${queuePaused}}:paused`, "1");

    const queuesFound = (await monitor.scan_queues()).filter((q) =>
      [queueEmpty, queuePartial, queuePaused].includes(q)
    ).sort();

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queues_found: queuesFound,
          stats_empty: await monitor.stats(queueEmpty),
          stats_partial: await monitor.stats(queuePartial),
          stats_paused: await monitor.stats(queuePaused),
          stats_many: await monitor.stats_many([queueEmpty, queuePartial, queuePaused]),
        },
        null,
        2
      )
    );
  } finally {
    await client.close();
    await seed.quit();
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
