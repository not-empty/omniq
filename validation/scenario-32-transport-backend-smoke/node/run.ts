import { OmniqClient, QueueMonitor } from "/workspace/omniq-node/src/index.ts";

const REDIS_HOST = process.env.REDIS_HOST ?? "omniq-redis";
const REDIS_PORT = 6379;
const REDIS_MODE = process.env.REDIS_MODE ?? "standalone";

async function main() {
  const queue = process.env.QUEUE ?? "validation-s32-node";

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const monitor = new QueueMonitor(client);

  try {
    await client.publish({
      queue,
      job_id: `${queue}-job-001`,
      payload: { kind: "transport-backend-smoke", backend: REDIS_MODE, sdk: "node" },
    });

    const reserved = await client.reserve({ queue });
    if (!reserved || (reserved as any).status !== "JOB") {
      throw new Error(`unexpected reserve response: ${JSON.stringify(reserved)}`);
    }

    const queuesFound = (await monitor.scan_queues()).filter((q) => q === queue).sort();
    if (JSON.stringify(queuesFound) !== JSON.stringify([queue])) {
      throw new Error(`unexpected discovered queues: ${JSON.stringify(queuesFound)}`);
    }

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          backend: REDIS_MODE,
          queue,
          reserve_status: (reserved as any).status,
          reserved_job_id: (reserved as any).job_id,
          queues_found: queuesFound,
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
