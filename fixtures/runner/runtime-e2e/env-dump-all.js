// Check all four MEW_TRANSFORM_* env vars.
// Writes "absent" for each if not set, or the value if leaked.
const { writeFileSync } = require("fs");
const vars = [
  "MEW_TRANSFORM_ENDPOINT",
  "MEW_TRANSFORM_TOKEN",
  "MEW_TRANSFORM_OPTIONS",
  "MEW_TRANSFORM_OPTS_DIGEST",
];
const results = vars.map((v) => v + "=" + (process.env[v] || "absent"));
writeFileSync("output.txt", results.join("\n"));
