try {
  require('pkg-b');
  process.exit(2);
} catch {
  process.exit(0);
}
