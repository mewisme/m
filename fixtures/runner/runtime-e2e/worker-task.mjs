// ESM Worker task — checks MEW_TRANSFORM_* in worker's process.env.
import { parentPort } from "node:worker_threads";

const vars = [
  "MEW_TRANSFORM_ENDPOINT",
  "MEW_TRANSFORM_TOKEN",
  "MEW_TRANSFORM_OPTIONS",
  "MEW_TRANSFORM_OPTS_DIGEST",
];
const results = vars.map((v) => v + "=" + (process.env[v] || "absent")).join("\n");
parentPort.postMessage(results);
