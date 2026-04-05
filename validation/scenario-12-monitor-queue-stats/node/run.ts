import { OmniqClient, QueueMonitor } from "/workspace/omniq-node/src/index.ts";

async function reserveJob(client: OmniqClient, queue: string, nowMs: number) {
  const res = await client.reserve({ queue, now_ms_override: nowMs });
  if (!res || (res as any).status !== "JOB") {
    throw new Error(`unexpected reserve response: ${JSON.stringify(res)}`);
  }
  return res as any;
}

async function main() {
  const prefix = process.env.PREFIX ?? "validation-s12-node";
  const queueA = `${prefix}-paused`;
  const queueB = `${prefix}-mixed`;
  const baseNowMs = 1775290000000;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const monitor = new QueueMonitor(client);

  try {
    await client.publish({ queue: queueA, job_id: `${queueA}-job-001`, payload: { kind: "monitor", queue: "a" }, now_ms_override: baseNowMs + 1 });
    await client.pause({ queue: queueA });

    const completedJob = `${queueB}-completed-job-001`;
    const activeJob = `${queueB}-active-job-001`;
    const delayedJob = `${queueB}-delayed-job-001`;

    await client.publish({ queue: queueB, job_id: completedJob, payload: { kind: "monitor", slot: "completed" }, now_ms_override: baseNowMs + 2 });
    await client.publish({ queue: queueB, job_id: activeJob, payload: { kind: "monitor", slot: "active" }, now_ms_override: baseNowMs + 3 });
    await client.publish({ queue: queueB, job_id: delayedJob, payload: { kind: "monitor", slot: "delayed" }, due_ms: baseNowMs + 100_000, now_ms_override: baseNowMs + 4 });

    const completedRes = await reserveJob(client, queueB, baseNowMs + 100);
    const activeRes = await reserveJob(client, queueB, baseNowMs + 101);
    await client.ack_success({
      queue: queueB,
      job_id: completedRes.job_id,
      lease_token: completedRes.lease_token,
      now_ms_override: baseNowMs + 150,
    });
    void activeRes;

    const listQueues = await monitor.list_queues();
    const queuesFound = listQueues.filter((q) => q === queueA || q === queueB).sort();
    const statsA = await monitor.stats(queueA);
    const statsB = await monitor.stats(queueB);
    const statsMany = await monitor.stats_many([queueA, queueB]);

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queues_found: queuesFound,
          stats_a: statsA,
          stats_b: statsB,
          stats_many: statsMany,
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
