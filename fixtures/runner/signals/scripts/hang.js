process.on("SIGINT", () => process.exit(130));
process.on("SIGTERM", () => process.exit(143));
setInterval(() => {}, 1 << 30);
