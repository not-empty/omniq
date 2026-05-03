import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

const REDIS_HOST = process.env.REDIS_HOST ?? "omniq-redis";
const REDIS_PORT = 6379;
const REDIS_MODE = process.env.REDIS_MODE ?? "standalone";

async function reserveJob(client: OmniqClient, queue: string, nowMs: number) {
  const res = await client.reserve({ queue, now_ms_override: nowMs });
  if (!res || (res as any).status !== "JOB") {
    throw new Error(`unexpected reserve response: ${JSON.stringify(res)}`);
  }
  return res as any;
}

async function main() {
  const queue = process.env.QUEUE ?? "validation-s19-node";
  const baseNowMs = 1775360000000;
  const dueMs = baseNowMs + 5000;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({ queue, job_id: `${queue}-alpha-job-001`, payload: { kind: "grouped-promote-delayed", slot: "alpha-1" }, gid: "alpha", group_limit: 1, due_ms: dueMs, now_ms_override: baseNowMs + 1 });
    await client.publish({ queue, job_id: `${queue}-beta-job-001`, payload: { kind: "grouped-promote-delayed", slot: "beta-1" }, gid: "beta", group_limit: 1, due_ms: dueMs, now_ms_override: baseNowMs + 2 });

    const promotedCount = await client.promote_delayed({ queue, max_promote: 1000, now_ms_override: dueMs });

    const r = (client as any).ops.r;
    const alphaReadyAfterPromote = (await r.zscore(`{${queue}}:groups:ready`, "alpha")) !== null;
    const betaReadyAfterPromote = (await r.zscore(`{${queue}}:groups:ready`, "beta")) !== null;
    const statsRaw = await r.hgetall(`{${queue}}:stats`);
    const groupWaitingAfterPromote = Number(statsRaw.group_waiting ?? 0);

    const nextOne = await reserveJob(client, queue, dueMs + 1);
    const nextTwo = await reserveJob(client, queue, dueMs + 2);

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          promoted_count: promotedCount,
          alpha_ready_after_promote: alphaReadyAfterPromote,
          beta_ready_after_promote: betaReadyAfterPromote,
          group_waiting_after_promote: groupWaitingAfterPromote,
          next_job_ids: [nextOne.job_id, nextTwo.job_id],
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
