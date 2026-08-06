// Spawn a Worker thread and check that it cannot read transform credentials.
import { Worker, isMainThread } from "node:worker_threads";
import { writeFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));

if (isMainThread) {
  const worker = new Worker(join(__dirname, "worker-task.mjs"));
  worker.on("message", (msg) => {
    writeFileSync("output.txt", msg);
  });
  worker.on("error", (err) => {
    writeFileSync("output.txt", "worker-error:" + err.message);
  });
} else {
  // Not reached — worker-task.mjs is the worker entry point.
  const results = [
    "MEW_TRANSFORM_ENDPOINT=" + (process.env.MEW_TRANSFORM_ENDPOINT || "absent"),
    "MEW_TRANSFORM_TOKEN=" + (process.env.MEW_TRANSFORM_TOKEN || "absent"),
  ].join("\n");
  writeFileSync("output.txt", results);
}
