const fs = require("fs");
const path = require("path");

const keys = Object.keys(process.env)
  .filter((k) => k === "INIT_CWD" || k.startsWith("npm_"))
  .sort();
const lines = keys.map((k) => k + "=" + process.env[k]);
fs.writeFileSync(path.join(__dirname, "..", "env.out"), lines.join("\n") + "\n");
