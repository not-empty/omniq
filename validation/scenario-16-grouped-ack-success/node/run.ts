import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

async function reserveJob(client: OmniqClient, queue: string, nowMs: number) {
  const res = await client.reserve({ queue, now_ms_override: nowMs });
  if (!res || (res as any).status !== "JOB") {
    throw new Error(`unexpected reserve response: ${JSON.stringify(res)}`);
  }
  return res as any;
}

async function main() {
  const queue = process.env.QUEUE ?? "validation-s16-node";
  const baseNowMs = 1775330000000;
  const gid = "alpha";
  const firstJob = `${queue}-alpha-job-001`;
  const secondJob = `${queue}-alpha-job-002`;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({
      queue,
      job_id: firstJob,
      payload: { kind: "grouped-ack-success", slot: "first" },
      gid,
      group_limit: 1,
      now_ms_override: baseNowMs + 1,
    });
    await client.publish({
      queue,
      job_id: secondJob,
      payload: { kind: "grouped-ack-success", slot: "second" },
      gid,
      group_limit: 1,
      now_ms_override: baseNowMs + 2,
    });

    const first = await reserveJob(client, queue, baseNowMs + 100);
    await client.ack_success({
      queue,
      job_id: first.job_id,
      lease_token: first.lease_token,
      now_ms_override: baseNowMs + 150,
    });

    const r = (client as any).ops.r;
    const groupReadyAfterAck = (await r.zscore(`{${queue}}:groups:ready`, gid)) !== null;
    const groupInflightAfterAck = Number(await r.get(`{${queue}}:g:${gid}:inflight`) ?? 0);

    const second = await reserveJob(client, queue, baseNowMs + 151);

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          first_job_id: first.job_id,
          second_job_id: second.job_id,
          group_ready_after_ack: groupReadyAfterAck,
          group_inflight_after_ack: groupInflightAfterAck,
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
