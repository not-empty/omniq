import { spawn } from "node:child_process";
import Redis from "/workspace/omniq-node/node_modules/ioredis/built/index.js";
import { OmniqClient } from "/workspace/omniq-node/src/index.ts";

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function childMain() {
  const queue = process.env.QUEUE as string;
  const markerStarted = process.env.MARKER_STARTED as string;
  const markerDone = process.env.MARKER_DONE as string;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const marker = new Redis("redis://omniq-redis:6379/0");

  try {
    await client.consume({
      queue,
      drain: false,
      poll_interval_s: 0.02,
      promote_interval_s: 10.0,
      reap_interval_s: 10.0,
      handler: async () => {
        await marker.set(markerStarted, "1");
        await sleep(1500);
        await marker.set(markerDone, "1");
      },
    });
  } finally {
    await client.close();
    await marker.quit();
  }
}

async function parentMain() {
  const queue = process.env.QUEUE ?? "validation-s27-node";
  const baseNowMs = 1775440000000;
  const firstJob = `${queue}-job-001`;
  const secondJob = `${queue}-job-002`;
  const markerStarted = `{${queue}}:marker:started`;
  const markerDone = `{${queue}}:marker:done`;

  const client = await OmniqClient.create({
    redis_url: "redis://omniq-redis:6379/0",
    scriptsDir: "/workspace/omniq-node/src/core/scripts",
  });
  const inspect = new Redis("redis://omniq-redis:6379/0");

  try {
    await client.publish({ queue, job_id: firstJob, payload: { kind: "drain-false", slot: 1 }, now_ms_override: baseNowMs + 1 });
    await client.publish({ queue, job_id: secondJob, payload: { kind: "drain-false", slot: 2 }, now_ms_override: baseNowMs + 2 });

    const child = spawn("npx", ["tsx", __filename, "child"], {
      env: { ...process.env, QUEUE: queue, MARKER_STARTED: markerStarted, MARKER_DONE: markerDone },
      stdio: "inherit",
    });

    const deadline = Date.now() + 5000;
    while (Date.now() < deadline) {
      if ((await inspect.get(markerStarted)) === "1") {
        break;
      }
      await sleep(50);
    }

    child.kill("SIGINT");

    const exitCode = await new Promise<number>((resolve) => {
      const timer = setTimeout(() => {
        child.kill("SIGKILL");
      }, 5000);
      child.on("exit", (code, signal) => {
        clearTimeout(timer);
        resolve(code ?? (signal === "SIGINT" ? 130 : 1));
      });
    });

    const statsKey = `{${queue}}:stats`;
    console.log(
      JSON.stringify(
        {
          sdk: "node",
          queue,
          child_exit_code: exitCode,
          handler_started: (await inspect.get(markerStarted)) === "1",
          handler_done: (await inspect.get(markerDone)) === "1",
          first_job_state: String(await inspect.hget(`{${queue}}:job:${firstJob}`, "state") ?? ""),
          second_job_state: String(await inspect.hget(`{${queue}}:job:${secondJob}`, "state") ?? ""),
          stats: {
            waiting: Number(await inspect.hget(statsKey, "waiting") ?? 0),
            waiting_total: Number(await inspect.hget(statsKey, "waiting_total") ?? 0),
            active: Number(await inspect.hget(statsKey, "active") ?? 0),
            completed_kept: Number(await inspect.hget(statsKey, "completed_kept") ?? 0),
          },
        },
        null,
        2
      )
    );
  } finally {
    await client.close();
    await inspect.quit();
  }
}

if (process.argv[2] === "child") {
  childMain().catch((err) => {
    console.error(err);
    process.exit(1);
  });
} else {
  parentMain().catch((err) => {
    console.error(err);
    process.exit(1);
  });
}
