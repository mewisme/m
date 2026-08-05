const { writeFileSync } = require("fs");
writeFileSync("output.txt", process.env.MEW_TRANSFORM_ENDPOINT || "absent");
