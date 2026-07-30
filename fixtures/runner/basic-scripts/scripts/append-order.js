const fs = require("fs");
const path = require("path");

const marker = path.join(__dirname, "..", "order.txt");
const label = process.argv[2];
if (!label) {
  process.exit(1);
}
fs.appendFileSync(marker, label + "\n");
