import Redis from "/workspace/omniq-node/node_modules/ioredis/built/index.js";
import { OmniqClient, QueueMonitor } from "/workspace/omniq-node/src/index.ts";

async function main() {
  const prefix = process.env.PREFIX ?? "validation-s24-node";
  const queueEmpty = `${prefix}-empty`;
  const queuePartial = `${prefix}-partial`;
  const queuePaused = `${prefix}-paused`;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const monitor = new QueueMonitor(client);
  const seed = new Redis("redis://omniq-redis:6379/0");

  try {
    await seed.sadd("omniq:queues", `{${queueEmpty}}`, `{${queuePartial}}`, `{${queuePaused}}`);
    await seed.hset(`{${queuePartial}}:stats`, {
      waiting: "2",
      group_waiting: "1",
      active: "3",
      last_activity_ms: "1775410000001",
    });
    await seed.set(`{${queuePaused}}:paused`, "1");

    const queuesFound = (await monitor.list_queues()).filter((q) =>
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
