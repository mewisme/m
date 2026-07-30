const fs = require("fs");
const path = require("path");

const pkg = path.basename(path.dirname(__dirname));
const root = path.join(__dirname, "..", "..", "..", ".results");
fs.mkdirSync(root, { recursive: true });
fs.writeFileSync(path.join(root, pkg + ".done"), String(Date.now()));
