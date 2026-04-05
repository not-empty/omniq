import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

async function main() {
  const queue = process.env.QUEUE ?? "validation-s02-node";
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
        kind: "heartbeat",
        source: "validation",
        sdk: "node",
        value: 2,
      },
      timeout_ms: 30_000,
      max_attempts: 3,
      backoff_ms: 5_000,
    });

    const reserve = await client.reserve({ queue });
    if (!reserve || (reserve as any).status !== "JOB") {
      throw new Error(`unexpected reserve result: ${JSON.stringify(reserve)}`);
    }

    const initialLock = (reserve as any).lock_until_ms as number;
    const newLock = await client.heartbeat({
      queue,
      job_id: (reserve as any).job_id,
      lease_token: (reserve as any).lease_token,
    });

    let badError = "";
    try {
      await client.heartbeat({
        queue,
        job_id: (reserve as any).job_id,
        lease_token: "bad-token",
      });
    } catch (err) {
      badError = String((err as any)?.message ?? err ?? "");
    }

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          job_id: (reserve as any).job_id,
          initial_lock_until_ms: initialLock,
          heartbeat_lock_until_ms: newLock,
          heartbeat_extended: newLock >= initialLock,
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
