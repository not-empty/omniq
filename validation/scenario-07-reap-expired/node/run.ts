import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

async function main() {
  const queue = process.env.QUEUE ?? "validation-s07-node";
  const retryJobId = process.env.RETRY_JOB_ID ?? `${queue}-retry-job-001`;
  const failJobId = process.env.FAIL_JOB_ID ?? `${queue}-fail-job-001`;
  const baseNowMs = 1775260000000;
  const reapNowMs = baseNowMs + 31000;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({
      queue,
      job_id: retryJobId,
      payload: { kind: "reap-expired", mode: "retry", sdk: "node" },
      timeout_ms: 30_000,
      max_attempts: 3,
      backoff_ms: 5_000,
      now_ms_override: baseNowMs,
    });
    await client.publish({
      queue,
      job_id: failJobId,
      payload: { kind: "reap-expired", mode: "terminal", sdk: "node" },
      timeout_ms: 30_000,
      max_attempts: 1,
      backoff_ms: 5_000,
      now_ms_override: baseNowMs,
    });

    const r1 = await client.reserve({ queue, now_ms_override: baseNowMs });
    const r2 = await client.reserve({ queue, now_ms_override: baseNowMs });
    if (!r1 || !r2) {
      throw new Error("expected two reserved jobs");
    }

    const reaped = await client.reap_expired({
      queue,
      max_reap: 1000,
      now_ms_override: reapNowMs,
    });

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          reaped_count: reaped,
          retryable_job_id: retryJobId,
          terminal_job_id: failJobId,
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
