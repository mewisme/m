const fs = require("fs");
const argv = process.argv.slice(2);
fs.writeFileSync("argv.out", JSON.stringify(argv));
console.log(JSON.stringify(argv));
