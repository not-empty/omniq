import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

const REDIS_HOST = process.env.REDIS_HOST ?? "omniq-redis";
const REDIS_PORT = 6379;
const REDIS_MODE = process.env.REDIS_MODE ?? "standalone";

async function main() {
  const queue = process.env.QUEUE ?? "validation-s06-node";
  const jobId = process.env.JOB_ID ?? `${queue}-job-001`;
  const baseNowMs = 1775250000000;
  const dueMs = baseNowMs + 5000;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({
      queue,
      job_id: jobId,
      payload: {
        kind: "promote-delayed",
        source: "validation",
        sdk: "node",
        value: 6,
      },
      timeout_ms: 30_000,
      max_attempts: 3,
      backoff_ms: 5_000,
      due_ms: dueMs,
      now_ms_override: baseNowMs,
    });

    const promoted = await client.promote_delayed({
      queue,
      max_promote: 1000,
      now_ms_override: dueMs,
    });

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          job_id: jobId,
          scheduled_due_ms: dueMs,
          promoted_count: promoted,
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
