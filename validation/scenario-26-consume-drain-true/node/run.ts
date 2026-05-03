import Redis from "/workspace/omniq-node/node_modules/ioredis/built/index.js";
import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

const REDIS_HOST = process.env.REDIS_HOST ?? "omniq-redis";
const REDIS_PORT = 6379;
const REDIS_MODE = process.env.REDIS_MODE ?? "standalone";

function newRawRedis() {
  if (REDIS_MODE === "cluster") {
    return new (Redis as any).Cluster([{ host: REDIS_HOST, port: REDIS_PORT }]);
  }
  return new Redis({ host: REDIS_HOST, port: REDIS_PORT });
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function main() {
  const queue = process.env.QUEUE ?? "validation-s26-node";
  const baseNowMs = 1775430000000;
  const firstJob = `${queue}-job-001`;
  const secondJob = `${queue}-job-002`;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const inspect = newRawRedis();

  const handledJobIds: string[] = [];
  let sigSent = false;

  try {
    await client.publish({ queue, job_id: firstJob, payload: { kind: "drain-true", slot: 1 }, now_ms_override: baseNowMs + 1 });
    await client.publish({ queue, job_id: secondJob, payload: { kind: "drain-true", slot: 2 }, now_ms_override: baseNowMs + 2 });

    await client.consume({
      queue,
      drain: true,
      stop_on_ctrl_c: true,
      poll_interval_s: 0.02,
      promote_interval_s: 10.0,
      reap_interval_s: 10.0,
      handler: async (ctx) => {
        handledJobIds.push(ctx.job_id);
        if (ctx.job_id === firstJob && !sigSent) {
          sigSent = true;
          setTimeout(() => process.kill(process.pid, "SIGINT"), 100);
        }
        await sleep(750);
      },
    });

    const statsKey = `{${queue}}:stats`;
    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          handled_job_ids: handledJobIds,
          first_job_state: String(await inspect.hget(`{${queue}}:job:${firstJob}`, "state") ?? ""),
          second_job_state: String(await inspect.hget(`{${queue}}:job:${secondJob}`, "state") ?? ""),
          stats: {
            waiting: Number(await inspect.hget(statsKey, "waiting") ?? 0),
            waiting_total: Number(await inspect.hget(statsKey, "waiting_total") ?? 0),
            active: Number(await inspect.hget(statsKey, "active") ?? 0),
            completed_kept: Number(await inspect.hget(statsKey, "completed_kept") ?? 0),
          },
        },
        null,
        2
      )
    );
  } finally {
    await client.close();
    await inspect.quit();
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
