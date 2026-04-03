import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

async function reserveJob(client: OmniqClient, queue: string, nowMs: number) {
  const res = await client.reserve({ queue, now_ms_override: nowMs });
  if (!res || (res as any).status !== "JOB") {
    throw new Error(`unexpected reserve response: ${JSON.stringify(res)}`);
  }
  return res as any;
}

async function capture(fn: () => Promise<unknown>) {
  try {
    await fn();
    return "NO_ERROR";
  } catch (err) {
    return err instanceof Error ? err.message : String(err);
  }
}

async function main() {
  const queue = process.env.QUEUE ?? "validation-s15-node";
  const baseNowMs = 1775320000000;

  const jobId = `${queue}-job-001`;
  const delayedJob = `${queue}-delayed-001`;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    const invalidPublish = await capture(() =>
      client.publish({ queue, job_id: `${queue}-bad-publish`, payload: "raw-string" as any })
    );

    await client.publish({ queue, job_id: jobId, payload: { kind: "error-surface" }, now_ms_override: baseNowMs + 1 });
    await client.publish({ queue, job_id: delayedJob, payload: { kind: "error-surface", slot: "delayed" }, due_ms: baseNowMs + 100_000, now_ms_override: baseNowMs + 2 });

    const reserved = await reserveJob(client, queue, baseNowMs + 100);

    const tokenMismatch = await capture(() =>
      client.ack_success({
        queue,
        job_id: reserved.job_id,
        lease_token: "bad-token",
        now_ms_override: baseNowMs + 110,
      })
    );

    await (client as any).ops.r.zrem(`{${queue}}:active`, reserved.job_id);

    const notActive = await capture(() =>
      client.ack_success({
        queue,
        job_id: reserved.job_id,
        lease_token: reserved.lease_token,
        now_ms_override: baseNowMs + 112,
      })
    );

    const batchLimit = await capture(() =>
      client.retry_failed_batch({
        queue,
        job_ids: Array.from({ length: 101 }, (_, i) => `${queue}-x-${String(i).padStart(3, "0")}`),
        now_ms_override: baseNowMs + 120,
      })
    );

    const laneMismatch = await capture(() =>
      client.remove_job({
        queue,
        job_id: delayedJob,
        lane: "wait",
      })
    );

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          token_mismatch: tokenMismatch,
          not_active: notActive,
          batch_limit: batchLimit,
          invalid_publish: invalidPublish,
          lane_mismatch: laneMismatch,
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
