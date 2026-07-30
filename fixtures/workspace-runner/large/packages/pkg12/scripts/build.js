const fs = require("fs");
const path = require("path");
const pkg = require("../package.json").name;
fs.writeFileSync(path.join(__dirname, "..", "out.txt"), pkg);
console.log(pkg);