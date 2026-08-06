// Spawn a child process and check that it cannot read transform credentials.
const { spawnSync } = require("node:child_process");
const { writeFileSync } = require("node:fs");

// Inline the checker as a node -e script.
const result = spawnSync(process.execPath, ["-e", `
  var vars = ["MEW_TRANSFORM_ENDPOINT","MEW_TRANSFORM_TOKEN","MEW_TRANSFORM_OPTIONS"];
  var results = vars.map(function(v) { return v + "=" + (process.env[v] || "absent"); });
  process.stdout.write(results.join("\\n"));
`], { encoding: "utf8" });

if (result.error) {
  writeFileSync("output.txt", "child-error:" + result.error.message);
} else {
  writeFileSync("output.txt", result.stdout.trim());
}
