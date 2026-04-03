import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

async function reserveJob(client: OmniqClient, queue: string, nowMs: number) {
  const res = await client.reserve({ queue, now_ms_override: nowMs });
  if (!res || (res as any).status !== "JOB") {
    throw new Error(`unexpected reserve response: ${JSON.stringify(res)}`);
  }
  return res as any;
}

async function main() {
  const queue = process.env.QUEUE ?? "validation-s20-node";
  const baseNowMs = 1775370000000;

  const failedJob = `${queue}-failed-job-001`;
  const waitingJob = `${queue}-waiting-job-001`;
  const missingJob = `${queue}-missing-job-001`;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({ queue, job_id: failedJob, payload: { kind: "batch-retry-errors", slot: "failed" }, max_attempts: 1, now_ms_override: baseNowMs + 1 });
    await client.publish({ queue, job_id: waitingJob, payload: { kind: "batch-retry-errors", slot: "waiting" }, max_attempts: 3, now_ms_override: baseNowMs + 2 });

    const failedRes = await reserveJob(client, queue, baseNowMs + 100);
    await client.ack_fail({
      queue,
      job_id: failedRes.job_id,
      lease_token: failedRes.lease_token,
      error: "make failed",
      now_ms_override: baseNowMs + 150,
    });

    const batchRetryResults = await client.retry_failed_batch({
      queue,
      job_ids: [failedJob, missingJob, waitingJob],
      now_ms_override: baseNowMs + 200,
    });

    const retriedJobState = await (client as any).ops.r.hget(`{${queue}}:job:${failedJob}`, "state");

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          batch_retry_results: batchRetryResults,
          retried_job_state: retriedJobState,
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
