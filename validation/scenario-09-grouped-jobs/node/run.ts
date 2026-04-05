import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

async function reserveJob(client: OmniqClient, queue: string, nowMs: number) {
  const res = await client.reserve({ queue, now_ms_override: nowMs });
  if (!res || (res as any).status !== "JOB") {
    throw new Error(`unexpected reserve response: ${JSON.stringify(res)}`);
  }
  return res as any;
}

async function main() {
  const queue = process.env.QUEUE ?? "validation-s09-node";
  const baseNowMs = 1775270000000;
  const alphaFirst = `${queue}-alpha-job-001`;
  const alphaSecond = `${queue}-alpha-job-002`;
  const betaFirst = `${queue}-beta-job-001`;
  const ungrouped = `${queue}-ungrouped-job-001`;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({
      queue,
      job_id: alphaFirst,
      payload: { kind: "grouped-jobs", slot: "alpha-1", sdk: "node" },
      gid: "alpha",
      group_limit: 1,
      now_ms_override: baseNowMs + 1,
    });
    await client.publish({
      queue,
      job_id: alphaSecond,
      payload: { kind: "grouped-jobs", slot: "alpha-2", sdk: "node" },
      gid: "alpha",
      group_limit: 5,
      now_ms_override: baseNowMs + 2,
    });
    await client.publish({
      queue,
      job_id: betaFirst,
      payload: { kind: "grouped-jobs", slot: "beta-1", sdk: "node" },
      gid: "beta",
      group_limit: 1,
      now_ms_override: baseNowMs + 3,
    });
    await client.publish({
      queue,
      job_id: ungrouped,
      payload: { kind: "grouped-jobs", slot: "ungrouped-1", sdk: "node" },
      now_ms_override: baseNowMs + 4,
    });

    const first = await reserveJob(client, queue, baseNowMs + 100);
    const second = await reserveJob(client, queue, baseNowMs + 101);
    const third = await reserveJob(client, queue, baseNowMs + 102);
    const fourth = await client.reserve({ queue, now_ms_override: baseNowMs + 103 });

    await client.ack_success({
      queue,
      job_id: first.job_id,
      lease_token: first.lease_token,
      now_ms_override: baseNowMs + 200,
    });
    const fifth = await reserveJob(client, queue, baseNowMs + 201);

    const rawRedis = (client as any).ops.r;
    const groupLimitAlpha = await rawRedis.get(`{${queue}}:g:alpha:limit`);

    const fourthStatus = fourth === null ? "EMPTY" : (fourth as any)?.status ?? null;

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          group_limit_alpha: groupLimitAlpha,
          reserve_order: [
            { job_id: first.job_id, gid: first.gid },
            { job_id: second.job_id, gid: second.gid },
            { job_id: third.job_id, gid: third.gid },
          ],
          fourth_reserve_status: fourthStatus,
          fifth_reserve_job_id: fifth.job_id,
          fifth_reserve_gid: fifth.gid,
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
