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
  const queue = process.env.QUEUE ?? "validation-s28-node";
  const jobId = `${queue}-job-001`;
  const baseNowMs = 1775440000000;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const inspect = newRawRedis();

  const seen: Array<{ attempt: number; max_attempts: number; is_last_attempt: boolean }> = [];
  let sigSent = false;

  try {
    await client.publish({
      queue,
      job_id: jobId,
      payload: { kind: "consume-max-attempts", sdk: "node" },
      max_attempts: 3,
      backoff_ms: 100,
      timeout_ms: 30_000,
      now_ms_override: baseNowMs + 1,
    });

    await client.consume({
      queue,
      drain: true,
      stop_on_ctrl_c: true,
      poll_interval_s: 0.02,
      promote_interval_s: 0.05,
      reap_interval_s: 10.0,
      handler: async (ctx) => {
        const isLastAttempt = ctx.attempt >= ctx.max_attempts;
        seen.push({
          attempt: ctx.attempt,
          max_attempts: ctx.max_attempts,
          is_last_attempt: isLastAttempt,
        });

        if (!isLastAttempt) {
          throw new Error("Intentional failure before the last attempt");
        }

        if (!sigSent) {
          sigSent = true;
          setTimeout(() => process.kill(process.pid, "SIGINT"), 50);
        }
        await sleep(100);
      },
    });

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          job_id: jobId,
          seen,
          final_state: String(await inspect.hget(`{${queue}}:job:${jobId}`, "state") ?? ""),
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
