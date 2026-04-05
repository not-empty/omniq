import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

async function reserveJob(client: OmniqClient, queue: string, nowMs: number) {
  const res = await client.reserve({ queue, now_ms_override: nowMs });
  if (!res || (res as any).status !== "JOB") {
    throw new Error(`unexpected reserve response: ${JSON.stringify(res)}`);
  }
  return res as any;
}

async function main() {
  const queue = process.env.QUEUE ?? "validation-s18-node";
  const baseNowMs = 1775350000000;
  const reapNowMs = baseNowMs + 31_000;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({ queue, job_id: `${queue}-alpha-job-001`, payload: { kind: "grouped-reap-expired", slot: "alpha-1" }, gid: "alpha", group_limit: 1, max_attempts: 3, timeout_ms: 30000, backoff_ms: 5000, now_ms_override: baseNowMs + 1 });
    await client.publish({ queue, job_id: `${queue}-alpha-job-002`, payload: { kind: "grouped-reap-expired", slot: "alpha-2" }, gid: "alpha", group_limit: 1, max_attempts: 3, timeout_ms: 30000, backoff_ms: 5000, now_ms_override: baseNowMs + 2 });
    await client.publish({ queue, job_id: `${queue}-beta-job-001`, payload: { kind: "grouped-reap-expired", slot: "beta-1" }, gid: "beta", group_limit: 1, max_attempts: 1, timeout_ms: 30000, backoff_ms: 5000, now_ms_override: baseNowMs + 3 });
    await client.publish({ queue, job_id: `${queue}-beta-job-002`, payload: { kind: "grouped-reap-expired", slot: "beta-2" }, gid: "beta", group_limit: 1, max_attempts: 1, timeout_ms: 30000, backoff_ms: 5000, now_ms_override: baseNowMs + 4 });

    await reserveJob(client, queue, baseNowMs + 100);
    await reserveJob(client, queue, baseNowMs + 101);

    const reapedCount = await client.reap_expired({ queue, max_reap: 1000, now_ms_override: reapNowMs });

    const r = (client as any).ops.r;
    const alphaInflightAfterReap = Number(await r.get(`{${queue}}:g:alpha:inflight`) ?? 0);
    const betaInflightAfterReap = Number(await r.get(`{${queue}}:g:beta:inflight`) ?? 0);
    const alphaReadyAfterReap = (await r.zscore(`{${queue}}:groups:ready`, "alpha")) !== null;
    const betaReadyAfterReap = (await r.zscore(`{${queue}}:groups:ready`, "beta")) !== null;

    const nextOne = await reserveJob(client, queue, reapNowMs + 1);
    const nextTwo = await reserveJob(client, queue, reapNowMs + 2);

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          reaped_count: reapedCount,
          alpha_inflight_after_reap: alphaInflightAfterReap,
          beta_inflight_after_reap: betaInflightAfterReap,
          alpha_ready_after_reap: alphaReadyAfterReap,
          beta_ready_after_reap: betaReadyAfterReap,
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
