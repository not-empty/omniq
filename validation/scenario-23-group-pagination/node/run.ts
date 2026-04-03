import { OmniqClient, QueueMonitor } from "/workspace/omniq-node/src/index.ts";

async function main() {
  const queue = process.env.QUEUE ?? "validation-s23-node";
  const baseNowMs = 1775400000000;
  const gids = ["alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta"];

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const monitor = new QueueMonitor(client);

  try {
    for (let i = 0; i < gids.length; i += 1) {
      const gid = gids[i];
      await client.publish({
        queue,
        job_id: `${queue}-${gid}-job-001`,
        payload: { kind: "group-pagination", gid, slot: 1 },
        gid,
        group_limit: 1,
        now_ms_override: baseNowMs + i + 1,
      });
    }

    const page1 = await monitor.groups_ready({ queue, offset: 0, limit: 3 });
    const page2 = await monitor.groups_ready({ queue, offset: 3, limit: 3 });
    const scoredPage1 = await monitor.groups_ready_with_scores({ queue, offset: 0, limit: 3 });
    const scoredPage2 = await monitor.groups_ready_with_scores({ queue, offset: 3, limit: 3 });
    const status = await monitor.group_status({ queue, gids: ["alpha", "delta", "eta"], default_limit: 1 });

    const groupsReadyRaw = (await (client as any).ops.r.zrange(`{${queue}}:groups:ready`, 0, -1)).map((x: unknown) => String(x ?? ""));

    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          groups_ready_page_1: page1,
          groups_ready_page_2: page2,
          groups_ready_scored_page_1: scoredPage1,
          groups_ready_scored_page_2: scoredPage2,
          group_status: status,
          groups_ready_raw: groupsReadyRaw,
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
