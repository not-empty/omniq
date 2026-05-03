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
  const prefix = process.env.PREFIX ?? "validation-s30-node";
  const queueA = `${prefix}-alpha`;
  const queueB = `${prefix}.beta_2`;
  const pausedOnly = `${prefix}-paused-only`;
  const invalidColonKey = `${prefix}-bad:name:stats`;
  const invalidSpaceKey = `{${prefix} bad}:stats`;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const monitor = new QueueMonitor(client);
  const seed = newRawRedis();

  try {
    await seed.hset(`{${queueA}}:stats`, { waiting: "0" });
    await seed.hset(`{${queueB}}:stats`, { waiting: "1" });
    await seed.set(`{${pausedOnly}}:paused`, "1");
    await seed.hset(invalidColonKey, { waiting: "9" });
    await seed.hset(invalidSpaceKey, { waiting: "9" });

    const queuesFound = (await monitor.scan_queues()).filter((q) => q.startsWith(prefix)).sort();
    const statsManyAuto = (await monitor.stats_many())
      .map((row) => row.queue)
      .filter((q) => q.startsWith(prefix))
      .sort();
    const expected = [queueA, queueB].sort();

    if (JSON.stringify(queuesFound) !== JSON.stringify(expected)) {
      throw new Error(`unexpected discovered queues: ${JSON.stringify(queuesFound)}`);
    }
    if (JSON.stringify(statsManyAuto) !== JSON.stringify(expected)) {
      throw new Error(`unexpected stats_many() discovery: ${JSON.stringify(statsManyAuto)}`);
    }
    if (queuesFound.includes(pausedOnly)) {
      throw new Error("paused-only queue should not be discovered");
    }
    if (queuesFound.some((q) => q.includes("bad"))) {
      throw new Error("invalid sparse keys leaked into queue discovery");
    }

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queues_found: queuesFound,
          stats_many_auto: statsManyAuto,
          paused_only_discovered: queuesFound.includes(pausedOnly),
          invalid_discovered: queuesFound.some((q) => q.includes("bad")),
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
