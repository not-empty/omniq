import { OmniqClient, QueueMonitor } from "/workspace/omniq-node/src/index.ts";

const REDIS_HOST = process.env.REDIS_HOST ?? "omniq-redis";
const REDIS_PORT = 6379;

async function main() {
  const queue = process.env.QUEUE ?? "validation-s29-node";
  const invalidNames = ["", " bad", "bad ", "bad:name", "{manual-tag}", "bad/name", "bad\\name", "bad name"];

  const client = await OmniqClient.create({
    host: REDIS_HOST,
    port: REDIS_PORT,
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const monitor = new QueueMonitor(client);

  try {
    const validJobId = await client.publish({
      queue,
      job_id: `${queue}-job-001`,
      payload: { kind: "queue-name-validation", sdk: "node" },
    });

    const invalidResults = [];
    for (const name of invalidNames) {
      let publishRejected = false;
      let statsRejected = false;

      try {
        await client.publish({ queue: name, payload: { kind: "invalid" } as any });
      } catch {
        publishRejected = true;
      }

      try {
        await monitor.stats(name);
      } catch {
        statsRejected = true;
      }

      invalidResults.push({
        queue: name,
        publish_rejected: publishRejected,
        stats_rejected: statsRejected,
      });
    }

    if (!validJobId) {
      throw new Error("valid queue did not publish a job id");
    }
    if (!invalidResults.every((row) => row.publish_rejected && row.stats_rejected)) {
      throw new Error("invalid queue names were not rejected consistently");
    }

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          valid_job_id: validJobId,
          invalid_results: invalidResults,
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
