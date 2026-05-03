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

function toInt(value: unknown): number {
  const n = Number(value ?? 0);
  return Number.isFinite(n) ? Math.trunc(n) : 0;
}

async function main() {
  const queue = process.env.QUEUE ?? "validation-s21-node";
  const baseNowMs = 1775380000000;

  const waitJob = `${queue}-wait-job-001`;
  const groupedWaitJob = `${queue}-grouped-wait-job-001`;
  const activeJob = `${queue}-active-job-001`;
  const delayedJob = `${queue}-delayed-job-001`;
  const missingJob = `${queue}-missing-job-001`;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({ queue, job_id: activeJob, payload: { kind: "batch-remove-errors", slot: "active" }, max_attempts: 3, now_ms_override: baseNowMs + 1 });

    const activeRes = await reserveJob(client, queue, baseNowMs + 100);
    if (activeRes.job_id !== activeJob) {
      throw new Error(`expected active job ${activeJob}, got ${activeRes.job_id}`);
    }

    await client.publish({ queue, job_id: waitJob, payload: { kind: "batch-remove-errors", slot: "wait" }, max_attempts: 3, now_ms_override: baseNowMs + 2 });
    await client.publish({ queue, job_id: groupedWaitJob, payload: { kind: "batch-remove-errors", slot: "gwait" }, max_attempts: 3, gid: "alpha", group_limit: 1, now_ms_override: baseNowMs + 3 });
    await client.publish({ queue, job_id: delayedJob, payload: { kind: "batch-remove-errors", slot: "delayed" }, max_attempts: 3, due_ms: baseNowMs + 100_000, now_ms_override: baseNowMs + 4 });

    const batchRemoveResults = await client.remove_jobs_batch({
      queue,
      lane: "wait",
      job_ids: [waitJob, missingJob, groupedWaitJob, activeJob, delayedJob],
    });

    const rawRedis = (client as any).ops.r;
    const statsKey = `{${queue}}:stats`;

    const stats = {
      waiting: toInt(await rawRedis.hget(statsKey, "waiting")),
      group_waiting: toInt(await rawRedis.hget(statsKey, "group_waiting")),
      waiting_total: toInt(await rawRedis.hget(statsKey, "waiting_total")),
      active: toInt(await rawRedis.hget(statsKey, "active")),
      delayed: toInt(await rawRedis.hget(statsKey, "delayed")),
      groups_ready: toInt(await rawRedis.hget(statsKey, "groups_ready")),
    };

    const jobHashExists = {
      wait_job: toInt(await rawRedis.exists(`{${queue}}:job:${waitJob}`)),
      grouped_wait_job: toInt(await rawRedis.exists(`{${queue}}:job:${groupedWaitJob}`)),
      active_job: toInt(await rawRedis.exists(`{${queue}}:job:${activeJob}`)),
      delayed_job: toInt(await rawRedis.exists(`{${queue}}:job:${delayedJob}`)),
    };

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          batch_remove_results: batchRemoveResults,
          job_hash_exists: jobHashExists,
          stats,
          wait_len: toInt(await rawRedis.llen(`{${queue}}:wait`)),
          idx_wait: toInt(await rawRedis.zcard(`{${queue}}:idx:wait`)),
          group_wait_len: toInt(await rawRedis.llen(`{${queue}}:g:alpha:wait`)),
          groups_ready: toInt(await rawRedis.zcard(`{${queue}}:groups:ready`)),
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
