import { OmniqClient, QueueMonitor } from "/workspace/omniq-node/src/index.ts";

const REDIS_HOST = process.env.REDIS_HOST ?? "omniq-redis";
const REDIS_PORT = 6379;
const REDIS_MODE = process.env.REDIS_MODE ?? "standalone";

function jobIds(rows: Array<{ job_id: string }>): string[] {
  return rows.map((row) => row.job_id);
}

async function main() {
  const queue = process.env.QUEUE ?? "validation-s22-node";
  const baseNowMs = 1775390000000;

  const waitJobs = Array.from({ length: 5 }, (_, i) => `${queue}-wait-${String(i + 1).padStart(3, "0")}`);
  const delayedJobs = Array.from({ length: 5 }, (_, i) => `${queue}-delayed-${String(i + 1).padStart(3, "0")}`);

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const monitor = new QueueMonitor(client);

  try {
    for (let i = 0; i < waitJobs.length; i += 1) {
      await client.publish({
        queue,
        job_id: waitJobs[i],
        payload: { kind: "lane-pagination", lane: "wait", order: i + 1 },
        now_ms_override: baseNowMs + i + 1,
      });
    }

    for (let i = 0; i < delayedJobs.length; i += 1) {
      await client.publish({
        queue,
        job_id: delayedJobs[i],
        payload: { kind: "lane-pagination", lane: "delayed", order: i + 1 },
        due_ms: baseNowMs + 100_000 + i + 1,
        now_ms_override: baseNowMs + 100 + i + 1,
      });
    }

    const waitForwardPages = [
      await monitor.lane_page({ queue, lane: "wait", offset: 0, limit: 2, reverse: false }),
      await monitor.lane_page({ queue, lane: "wait", offset: 2, limit: 2, reverse: false }),
      await monitor.lane_page({ queue, lane: "wait", offset: 4, limit: 2, reverse: false }),
    ];
    const waitReversePages = [
      await monitor.lane_page({ queue, lane: "wait", offset: 0, limit: 2, reverse: true }),
      await monitor.lane_page({ queue, lane: "wait", offset: 2, limit: 2, reverse: true }),
      await monitor.lane_page({ queue, lane: "wait", offset: 4, limit: 2, reverse: true }),
    ];
    const delayedForwardPages = [
      await monitor.lane_page({ queue, lane: "delayed", offset: 0, limit: 2, reverse: false }),
      await monitor.lane_page({ queue, lane: "delayed", offset: 2, limit: 2, reverse: false }),
      await monitor.lane_page({ queue, lane: "delayed", offset: 4, limit: 2, reverse: false }),
    ];
    const delayedReversePages = [
      await monitor.lane_page({ queue, lane: "delayed", offset: 0, limit: 2, reverse: true }),
      await monitor.lane_page({ queue, lane: "delayed", offset: 2, limit: 2, reverse: true }),
      await monitor.lane_page({ queue, lane: "delayed", offset: 4, limit: 2, reverse: true }),
    ];

    const rawRedis = (client as any).ops.r;

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          stats: await monitor.stats(queue),
          wait_forward_pages: waitForwardPages.map(jobIds),
          wait_reverse_pages: waitReversePages.map(jobIds),
          delayed_forward_pages: delayedForwardPages.map(jobIds),
          delayed_reverse_pages: delayedReversePages.map(jobIds),
          idx_wait_raw: (await rawRedis.zrange(`{${queue}}:idx:wait`, 0, -1)).map((x: unknown) => String(x ?? "")),
          idx_delayed_raw: (await rawRedis.zrange(`{${queue}}:idx:delayed`, 0, -1)).map((x: unknown) => String(x ?? "")),
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
