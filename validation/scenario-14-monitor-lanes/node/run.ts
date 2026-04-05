import { OmniqClient, QueueMonitor } from "/workspace/omniq-node/src/index.ts";

async function reserveJob(client: OmniqClient, queue: string, nowMs: number) {
  const res = await client.reserve({ queue, now_ms_override: nowMs });
  if (!res || (res as any).status !== "JOB") {
    throw new Error(`unexpected reserve response: ${JSON.stringify(res)}`);
  }
  return res as any;
}

async function main() {
  const queue = process.env.QUEUE ?? "validation-s14-node";
  const baseNowMs = 1775310000000;

  const waitKeep = `${queue}-wait-keep-001`;
  const waitMissing = `${queue}-wait-missing-001`;
  const activeJob = `${queue}-active-001`;
  const delayedJob = `${queue}-delayed-001`;
  const completedJob = `${queue}-completed-001`;
  const failedJob = `${queue}-failed-001`;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const monitor = new QueueMonitor(client);

  try {
    await client.publish({ queue, job_id: completedJob, payload: { kind: "monitor-lanes", slot: "completed" }, now_ms_override: baseNowMs + 1 });
    await client.publish({ queue, job_id: activeJob, payload: { kind: "monitor-lanes", slot: "active" }, now_ms_override: baseNowMs + 2 });
    await client.publish({ queue, job_id: failedJob, payload: { kind: "monitor-lanes", slot: "failed" }, max_attempts: 1, now_ms_override: baseNowMs + 3 });
    await client.publish({ queue, job_id: delayedJob, payload: { kind: "monitor-lanes", slot: "delayed" }, due_ms: baseNowMs + 100_000, now_ms_override: baseNowMs + 4 });
    await client.publish({ queue, job_id: waitKeep, payload: { kind: "monitor-lanes", slot: "wait-keep" }, now_ms_override: baseNowMs + 5 });
    await client.publish({ queue, job_id: waitMissing, payload: { kind: "monitor-lanes", slot: "wait-missing" }, now_ms_override: baseNowMs + 6 });

    const completedRes = await reserveJob(client, queue, baseNowMs + 100);
    const activeRes = await reserveJob(client, queue, baseNowMs + 101);
    const failedRes = await reserveJob(client, queue, baseNowMs + 102);

    await client.ack_success({ queue, job_id: completedRes.job_id, lease_token: completedRes.lease_token, now_ms_override: baseNowMs + 150 });
    await client.ack_fail({ queue, job_id: failedRes.job_id, lease_token: failedRes.lease_token, error: "terminal failure", now_ms_override: baseNowMs + 151 });

    await (client as any).ops.r.del(`{${queue}}:job:${waitMissing}`);
    void activeRes;

    const waitPage = await monitor.lane_page({ queue, lane: "wait", offset: 0, limit: 10, reverse: false });
    const waitPageReverse = await monitor.lane_page({ queue, lane: "wait", offset: 0, limit: 10, reverse: true });
    const findWait = await monitor.find_jobs({ queue, lane: "wait", job_ids: [waitKeep, waitMissing] });
    const getExisting = await monitor.get_job(queue, activeJob);
    const getMissing = await monitor.get_job(queue, waitMissing);
    const overview = await monitor.overview(queue, 10);

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          wait_page: waitPage,
          wait_page_reverse: waitPageReverse,
          find_wait: findWait,
          get_existing: getExisting,
          get_missing: getMissing,
          overview,
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
