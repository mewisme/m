const fs = require("fs");
const msg = process.argv[2] || "";
fs.writeFileSync("script.out", msg);
process.stdout.write(msg);
