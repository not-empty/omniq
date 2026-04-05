import { OmniqClient, QueueMonitor } from "/workspace/omniq-node/src/index.ts";

async function main() {
  const queue = process.env.QUEUE ?? "validation-s13-node";
  const baseNowMs = 1775300000000;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const monitor = new QueueMonitor(client);

  try {
    await client.publish({ queue, job_id: `${queue}-alpha-job-001`, payload: { kind: "monitor-groups", slot: "alpha-1" }, gid: "alpha", group_limit: 2, now_ms_override: baseNowMs + 1 });
    await client.publish({ queue, job_id: `${queue}-alpha-job-002`, payload: { kind: "monitor-groups", slot: "alpha-2" }, gid: "alpha", group_limit: 2, now_ms_override: baseNowMs + 2 });
    await client.publish({ queue, job_id: `${queue}-beta-job-001`, payload: { kind: "monitor-groups", slot: "beta-1" }, gid: "beta", group_limit: 1, now_ms_override: baseNowMs + 3 });
    await client.publish({ queue, job_id: `${queue}-gamma-job-001`, payload: { kind: "monitor-groups", slot: "gamma-1" }, gid: "gamma", group_limit: 1, now_ms_override: baseNowMs + 4 });
    await client.publish({ queue, job_id: `${queue}-delta-job-001`, payload: { kind: "monitor-groups", slot: "delta-1" }, gid: "delta", group_limit: 1, now_ms_override: baseNowMs + 5 });

    const first = await client.reserve({ queue, now_ms_override: baseNowMs + 100 });
    if (!first || (first as any).status !== "JOB") {
      throw new Error(`unexpected reserve response: ${JSON.stringify(first)}`);
    }

    const gids = ["alpha", "beta", "gamma", "delta"];
    const groupsReadyPage = await monitor.groups_ready({ queue, offset: 0, limit: 2 });
    const groupsReadyAll = await monitor.groups_ready_with_scores({ queue, offset: 0, limit: 10 });
    const groupStatus = await monitor.group_status({ queue, gids, default_limit: 1 });

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          groups_ready_page: groupsReadyPage,
          groups_ready_all: groupsReadyAll,
          group_status: groupStatus,
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
