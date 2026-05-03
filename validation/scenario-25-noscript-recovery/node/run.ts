import Redis from "/workspace/omniq-node/node_modules/ioredis/built/index.js";
import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

const REDIS_HOST = process.env.REDIS_HOST ?? "omniq-redis";
const REDIS_PORT = 6379;
const REDIS_MODE = process.env.REDIS_MODE ?? "standalone";

function newRawRedis() {
  if (REDIS_MODE === "cluster") {
    return new (Redis as any).Cluster([{ host: REDIS_HOST, port: REDIS_PORT }]);
  }
  return new Redis({ host: REDIS_HOST, port: REDIS_PORT });
}

async function reserveJob(client: OmniqClient, queue: string, nowMs: number) {
  const res = await client.reserve({ queue, now_ms_override: nowMs });
  if (!res || (res as any).status !== "JOB") {
    throw new Error(`unexpected reserve response: ${JSON.stringify(res)}`);
  }
  return res as any;
}

async function scriptFlush(seed: any) {
  await seed.script("flush");
}

async function main() {
  const queue = process.env.QUEUE ?? "validation-s25-node";
  const baseNowMs = 1775420000000;

  const publishJob = `${queue}-job-001`;
  const delayedJob = `${queue}-delayed-001`;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const seed = newRawRedis();

  try {
    await scriptFlush(seed);
    const publishedJobId = await client.publish({
      queue,
      job_id: publishJob,
      payload: { kind: "noscript-recovery", slot: "publish" },
      now_ms_override: baseNowMs + 1,
    });

    await scriptFlush(seed);
    const reserved = await reserveJob(client, queue, baseNowMs + 100);

    await scriptFlush(seed);
    const heartbeatLockUntilMs = await client.heartbeat({
      queue,
      job_id: reserved.job_id,
      lease_token: reserved.lease_token,
      now_ms_override: baseNowMs + 110,
    });

    await scriptFlush(seed);
    await client.ack_success({
      queue,
      job_id: reserved.job_id,
      lease_token: reserved.lease_token,
      now_ms_override: baseNowMs + 120,
    });

    await scriptFlush(seed);
    const delayedJobId = await client.publish({
      queue,
      job_id: delayedJob,
      payload: { kind: "noscript-recovery", slot: "delayed" },
      due_ms: baseNowMs + 500,
      now_ms_override: baseNowMs + 2,
    });

    await scriptFlush(seed);
    const promotedCount = await client.promote_delayed({
      queue,
      max_promote: 10,
      now_ms_override: baseNowMs + 600,
    });

    const rawRedis = (client as any).ops.r;
    const completedState = String(await rawRedis.hget(`{${queue}}:job:${publishJob}`, "state") ?? "");
    const promotedState = String(await rawRedis.hget(`{${queue}}:job:${delayedJob}`, "state") ?? "");

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          published_job_id: publishedJobId,
          reserved_job_id: reserved.job_id,
          heartbeat_lock_until_ms: heartbeatLockUntilMs,
          completed_state: completedState,
          delayed_job_id: delayedJobId,
          promoted_count: promotedCount,
          promoted_state: promotedState,
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
