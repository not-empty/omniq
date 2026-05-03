import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

const REDIS_HOST = process.env.REDIS_HOST ?? "omniq-redis";
const REDIS_PORT = 6379;
const REDIS_MODE = process.env.REDIS_MODE ?? "standalone";

async function main() {
  const queue = process.env.QUEUE ?? "validation-s08-node";
  const firstJob = `${queue}-job-001`;
  const secondJob = `${queue}-job-002`;

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({ queue, job_id: firstJob, payload: { kind: "pause-resume", n: 1 } });
    await client.publish({ queue, job_id: secondJob, payload: { kind: "pause-resume", n: 2 } });

    const pausedBefore = await client.is_paused({ queue });
    const first = await client.reserve({ queue });
    if (!first || (first as any).status !== "JOB") {
      throw new Error(`unexpected first reserve: ${JSON.stringify(first)}`);
    }

    await client.pause({ queue });
    const pausedAfterPause = await client.is_paused({ queue });
    const pausedReserve = await client.reserve({ queue });

    await client.resume({ queue });
    const pausedAfterResume = await client.is_paused({ queue });
    const second = await client.reserve({ queue });
    if (!second || (second as any).status !== "JOB") {
      throw new Error(`unexpected second reserve: ${JSON.stringify(second)}`);
    }

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          paused_before: pausedBefore,
          paused_after_pause: pausedAfterPause,
          paused_after_resume: pausedAfterResume,
          paused_reserve_status: (pausedReserve as any)?.status ?? null,
          first_reserved_job_id: (first as any).job_id,
          second_reserved_job_id: (second as any).job_id,
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
