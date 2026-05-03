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
  const queue = process.env.QUEUE ?? "validation-s10-node";
  const baseNowMs = 1775280000000;

  const activeJob = `${queue}-active-job-001`;
  const singleRetryJob = `${queue}-single-retry-job-001`;
  const batchRetryJob = `${queue}-batch-retry-job-001`;
  const waitingRemoveJob = `${queue}-waiting-remove-job-001`;
  const delayedRemoveJob = `${queue}-delayed-remove-job-001`;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({ queue, job_id: activeJob, payload: { kind: "admin", slot: "active" }, max_attempts: 3, now_ms_override: baseNowMs + 1 });
    await client.publish({ queue, job_id: singleRetryJob, payload: { kind: "admin", slot: "single-retry" }, max_attempts: 1, now_ms_override: baseNowMs + 2 });
    await client.publish({ queue, job_id: batchRetryJob, payload: { kind: "admin", slot: "batch-retry" }, max_attempts: 1, now_ms_override: baseNowMs + 3 });
    await client.publish({ queue, job_id: waitingRemoveJob, payload: { kind: "admin", slot: "waiting-remove" }, max_attempts: 3, now_ms_override: baseNowMs + 4 });
    await client.publish({ queue, job_id: delayedRemoveJob, payload: { kind: "admin", slot: "delayed-remove" }, max_attempts: 3, due_ms: baseNowMs + 100_000, now_ms_override: baseNowMs + 5 });

    const activeRes = await reserveJob(client, queue, baseNowMs + 100);
    const singleFailedRes = await reserveJob(client, queue, baseNowMs + 101);
    const batchFailedRes = await reserveJob(client, queue, baseNowMs + 102);

    await client.ack_fail({
      queue,
      job_id: singleFailedRes.job_id,
      lease_token: singleFailedRes.lease_token,
      error: "single retry setup",
      now_ms_override: baseNowMs + 150,
    });
    await client.ack_fail({
      queue,
      job_id: batchFailedRes.job_id,
      lease_token: batchFailedRes.lease_token,
      error: "batch retry setup",
      now_ms_override: baseNowMs + 151,
    });

    await client.retry_failed({ queue, job_id: singleRetryJob, now_ms_override: baseNowMs + 200 });

    const batchRetryResults = await client.retry_failed_batch({
      queue,
      job_ids: [batchRetryJob, waitingRemoveJob],
      now_ms_override: baseNowMs + 201,
    });

    let removeActiveError = "NO_ERROR";
    try {
      await client.remove_job({ queue, job_id: activeJob, lane: "wait" });
    } catch (err) {
      removeActiveError = err instanceof Error ? err.message : String(err);
    }

    const batchRemoveResults = await client.remove_jobs_batch({
      queue,
      lane: "wait",
      job_ids: [waitingRemoveJob, delayedRemoveJob],
    });

    const delayedRemoveResult = await client.remove_job({
      queue,
      job_id: delayedRemoveJob,
      lane: "delayed",
    });

    const rawRedis = (client as any).ops.r;
    const singleRetryState = String(await rawRedis.hget(`{${queue}}:job:${singleRetryJob}`, "state") ?? "");
    const singleRetryAttempt = Number(await rawRedis.hget(`{${queue}}:job:${singleRetryJob}`, "attempt") ?? 0);

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          single_retry_state: singleRetryState,
          single_retry_attempt: singleRetryAttempt,
          batch_retry_results: batchRetryResults,
          remove_active_error: removeActiveError,
          batch_remove_results: batchRemoveResults,
          delayed_remove_result: delayedRemoveResult,
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
