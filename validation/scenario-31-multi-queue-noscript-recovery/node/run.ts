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

async function scriptFlush(seed: any) {
  await seed.script("flush");
}

async function reserveJob(client: OmniqClient, queue: string, nowMs: number) {
  const res = await client.reserve({ queue, now_ms_override: nowMs });
  if (!res || (res as any).status !== "JOB") {
    throw new Error(`unexpected reserve response: ${JSON.stringify(res)}`);
  }
  return res as any;
}

async function main() {
  const queuePrefix = process.env.QUEUE ?? "validation-s31-node";
  const queueA = `${queuePrefix}-a`;
  const queueB = `${queuePrefix}-b`;
  const baseNowMs = 1775450000000;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const monitor = new QueueMonitor(client);
  const seed = newRawRedis();

  try {
    await scriptFlush(seed);
    await client.publish({
      queue: queueA,
      job_id: `${queueA}-job-001`,
      payload: { kind: "multi-queue-noscript", queue: "a" },
      now_ms_override: baseNowMs + 1,
    });

    await scriptFlush(seed);
    await client.publish({
      queue: queueB,
      job_id: `${queueB}-job-001`,
      payload: { kind: "multi-queue-noscript", queue: "b" },
      now_ms_override: baseNowMs + 2,
    });

    await scriptFlush(seed);
    const reservedA = await reserveJob(client, queueA, baseNowMs + 100);

    await scriptFlush(seed);
    await client.ack_success({
      queue: queueA,
      job_id: reservedA.job_id,
      lease_token: reservedA.lease_token,
      now_ms_override: baseNowMs + 110,
    });

    await scriptFlush(seed);
    const reservedB = await reserveJob(client, queueB, baseNowMs + 120);

    await scriptFlush(seed);
    const heartbeatB = await client.heartbeat({
      queue: queueB,
      job_id: reservedB.job_id,
      lease_token: reservedB.lease_token,
      now_ms_override: baseNowMs + 130,
    });

    await scriptFlush(seed);
    await client.ack_success({
      queue: queueB,
      job_id: reservedB.job_id,
      lease_token: reservedB.lease_token,
      now_ms_override: baseNowMs + 140,
    });

    const rawRedis = (client as any).ops.r;
    const queuesFound = (await monitor.scan_queues()).filter((q) => [queueA, queueB].includes(q)).sort();
    const queueAState = String((await rawRedis.hget(`{${queueA}}:job:${queueA}-job-001`, "state")) ?? "");
    const queueBState = String((await rawRedis.hget(`{${queueB}}:job:${queueB}-job-001`, "state")) ?? "");

    if (JSON.stringify(queuesFound) !== JSON.stringify([queueA, queueB].sort())) {
      throw new Error(`unexpected discovered queues: ${JSON.stringify(queuesFound)}`);
    }
    if (queueAState !== "completed" || queueBState !== "completed") {
      throw new Error("multi-queue NOSCRIPT flow did not complete both jobs");
    }
    if (heartbeatB <= 0) {
      throw new Error("heartbeat did not extend queue B lease");
    }

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queues_found: queuesFound,
          queue_a_state: queueAState,
          queue_b_state: queueBState,
          heartbeat_b: heartbeatB,
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
