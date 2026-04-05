import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

async function main() {
  const queue = process.env.QUEUE ?? "validation-s05-node";
  const jobId = process.env.JOB_ID ?? `${queue}-job-001`;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  try {
    await client.publish({
      queue,
      job_id: jobId,
      payload: {
        kind: "ack-fail-terminal",
        source: "validation",
        sdk: "node",
        value: 5,
      },
      timeout_ms: 30_000,
      max_attempts: 1,
      backoff_ms: 5_000,
    });

    const reserve = await client.reserve({ queue });
    if (!reserve || (reserve as any).status !== "JOB") {
      throw new Error(`unexpected reserve result: ${JSON.stringify(reserve)}`);
    }

    let badError = "";
    try {
      await client.ack_fail({
        queue,
        job_id: (reserve as any).job_id,
        lease_token: "bad-token",
        error: "boom-terminal",
      });
    } catch (err) {
      badError = String((err as any)?.message ?? err ?? "");
    }

    const [status, dueMs] = await client.ack_fail({
      queue,
      job_id: (reserve as any).job_id,
      lease_token: (reserve as any).lease_token,
      error: "boom-terminal",
    });

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          job_id: (reserve as any).job_id,
          ack_fail_status: status,
          due_ms: dueMs,
          invalid_token_error: badError,
          invalid_token_contains_token_mismatch: badError.includes("TOKEN_MISMATCH"),
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
