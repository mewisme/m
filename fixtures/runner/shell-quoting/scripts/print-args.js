const fs = require("fs");
const path = require("path");

const lines = process.argv.slice(2);
fs.writeFileSync(path.join(__dirname, "..", "args.out"), lines.join("\n") + (lines.length ? "\n" : ""));
