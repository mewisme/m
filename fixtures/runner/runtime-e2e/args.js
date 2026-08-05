const { writeFileSync } = require("fs");
writeFileSync("output.txt", JSON.stringify(process.argv.slice(2)) + "\n");
