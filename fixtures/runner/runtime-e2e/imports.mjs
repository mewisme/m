import { greeting } from "./imports-lib.mjs";
import { writeFileSync } from "node:fs";
writeFileSync("output.txt", greeting + "\n");
