const fs = require("fs");
const path = require("path");

const pkg = path.basename(path.dirname(__dirname));
const root = path.join(__dirname, "..", "..", "..", ".results");
const upstream = path.join(root, "lib.done");
if (!fs.existsSync(upstream)) {
  console.error("missing upstream marker: lib.done");
  process.exit(1);
}
fs.mkdirSync(root, { recursive: true });
fs.writeFileSync(path.join(root, pkg + ".done"), String(Date.now()));
