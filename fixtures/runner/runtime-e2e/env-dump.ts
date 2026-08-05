import { writeFileSync } from "node:fs";
writeFileSync("output.txt", process.env.MEW_TRANSFORM_ENDPOINT || "absent");
