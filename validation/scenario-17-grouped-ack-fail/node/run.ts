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
  const queue = process.env.QUEUE ?? "validation-s17-node";
  const baseNowMs = 1775340000000;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({ queue, job_id: `${queue}-alpha-job-001`, payload: { kind: "grouped-ack-fail", slot: "alpha-1" }, gid: "alpha", group_limit: 1, max_attempts: 3, backoff_ms: 5000, now_ms_override: baseNowMs + 1 });
    await client.publish({ queue, job_id: `${queue}-alpha-job-002`, payload: { kind: "grouped-ack-fail", slot: "alpha-2" }, gid: "alpha", group_limit: 1, max_attempts: 3, backoff_ms: 5000, now_ms_override: baseNowMs + 2 });
    await client.publish({ queue, job_id: `${queue}-beta-job-001`, payload: { kind: "grouped-ack-fail", slot: "beta-1" }, gid: "beta", group_limit: 1, max_attempts: 1, backoff_ms: 5000, now_ms_override: baseNowMs + 3 });
    await client.publish({ queue, job_id: `${queue}-beta-job-002`, payload: { kind: "grouped-ack-fail", slot: "beta-2" }, gid: "beta", group_limit: 1, max_attempts: 1, backoff_ms: 5000, now_ms_override: baseNowMs + 4 });

    const alphaFirst = await reserveJob(client, queue, baseNowMs + 100);
    const betaFirst = await reserveJob(client, queue, baseNowMs + 101);

    const alphaFail = await client.ack_fail({ queue, job_id: alphaFirst.job_id, lease_token: alphaFirst.lease_token, error: "retryable grouped fail", now_ms_override: baseNowMs + 150 });
    const betaFail = await client.ack_fail({ queue, job_id: betaFirst.job_id, lease_token: betaFirst.lease_token, error: "terminal grouped fail", now_ms_override: baseNowMs + 151 });

    const r = (client as any).ops.r;
    const alphaInflightAfterFail = Number(await r.get(`{${queue}}:g:alpha:inflight`) ?? 0);
    const betaInflightAfterFail = Number(await r.get(`{${queue}}:g:beta:inflight`) ?? 0);
    const alphaReadyAfterFail = (await r.zscore(`{${queue}}:groups:ready`, "alpha")) !== null;
    const betaReadyAfterFail = (await r.zscore(`{${queue}}:groups:ready`, "beta")) !== null;

    const nextOne = await reserveJob(client, queue, baseNowMs + 152);
    const nextTwo = await reserveJob(client, queue, baseNowMs + 153);

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          alpha_fail_status: alphaFail[0],
          beta_fail_status: betaFail[0],
          alpha_inflight_after_fail: alphaInflightAfterFail,
          beta_inflight_after_fail: betaInflightAfterFail,
          alpha_ready_after_fail: alphaReadyAfterFail,
          beta_ready_after_fail: betaReadyAfterFail,
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
