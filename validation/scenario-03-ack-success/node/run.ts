import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

const REDIS_HOST = process.env.REDIS_HOST ?? "omniq-redis";
const REDIS_PORT = 6379;
const REDIS_MODE = process.env.REDIS_MODE ?? "standalone";

async function main() {
  const queue = process.env.QUEUE ?? "validation-s03-node";
  const jobId = process.env.JOB_ID ?? `${queue}-job-001`;

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
        kind: "ack-success",
        source: "validation",
        sdk: "node",
        value: 3,
      },
      timeout_ms: 30_000,
      max_attempts: 3,
      backoff_ms: 5_000,
    });

    const reserve = await client.reserve({ queue });
    if (!reserve || (reserve as any).status !== "JOB") {
      throw new Error(`unexpected reserve result: ${JSON.stringify(reserve)}`);
    }

    let badError = "";
    try {
      await client.ack_success({
        queue,
        job_id: (reserve as any).job_id,
        lease_token: "bad-token",
      });
    } catch (err) {
      badError = String((err as any)?.message ?? err ?? "");
    }

    await client.ack_success({
      queue,
      job_id: (reserve as any).job_id,
      lease_token: (reserve as any).lease_token,
    });

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          job_id: (reserve as any).job_id,
          ack_success_ok: true,
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
