import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

async function main() {
  const queue = process.env.QUEUE ?? "validation-basic-node";
  const jobId = process.env.JOB_ID ?? `${queue}-job-001`;
  const payload = {
    kind: "basic-reserve",
    source: "validation",
    sdk: "node",
    value: 1,
  };

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });

  let invalidPublishRejected = false;

  try {
    try {
      await client.publish({
        queue,
        payload: "raw-string" as unknown as Record<string, unknown>,
      });
    } catch {
      invalidPublishRejected = true;
    }

    const publishedJobId = await client.publish({
      queue,
      job_id: jobId,
      payload,
      timeout_ms: 30_000,
      max_attempts: 3,
      backoff_ms: 5_000,
    });

    const reserve = await client.reserve({ queue });

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          invalid_publish_rejected: invalidPublishRejected,
          job_id: publishedJobId,
          reserve:
            reserve === null
              ? null
              : {
                  status: (reserve as any).status ?? null,
                  job_id: (reserve as any).job_id ?? null,
                  payload: (reserve as any).payload ?? null,
                  attempt: (reserve as any).attempt ?? null,
                  max_attempts: (reserve as any).max_attempts ?? null,
                  gid: (reserve as any).gid ?? null,
                  lease_token_present: Boolean((reserve as any).lease_token ?? ""),
                },
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
