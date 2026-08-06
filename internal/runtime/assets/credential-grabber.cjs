// Mew credential grabber — runs before any user preload.
// Node processes --require from left to right. Mew places this
// grabber first so it captures transform credentials from process.env
// and strips them before user code executes.
//
// This module runs twice in Node's two-phase startup:
//   1. Main thread (isMainThread=true): captures env, writes creds to
//      a temp file, deletes env vars. User --require preloads that
//      follow see empty MEW_TRANSFORM_* vars.
//   2. Loader context (isMainThread=false): reads creds from the temp
//      file, deletes the file and file-path env var, exports values.
//
// The ts-loader (loaded via module.register) uses createRequire to
// load this module and receives the loader-context copy of the exports.
'use strict';

const { isMainThread } = require('node:worker_threads');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

// Temp file path — deterministic within this process so both the main
// thread and the loader context agree on the location without an env var.
const credsFile = path.join(os.tmpdir(), '.mew-creds-' + process.pid + '.json');

if (isMainThread) {
  // ── Main thread ────────────────────────────────────────────────
  // Capture credentials from process.env and write to temp file
  // before any user --require or --import preload executes.
  const endpoint = process.env.MEW_TRANSFORM_ENDPOINT || null;
  const token = process.env.MEW_TRANSFORM_TOKEN || null;
  const options = process.env.MEW_TRANSFORM_OPTIONS || '{}';
  const optsDigest = process.env.MEW_TRANSFORM_OPTS_DIGEST || '';

  if (endpoint && token) {
    try {
      fs.writeFileSync(credsFile, JSON.stringify({
        endpoint, token, options, optsDigest,
      }), { mode: 0o600 });
    } catch (_) {
      // Write failure: exports will be null, ts-loader will fail
      // with "no endpoint or token" — safe, no credential leak.
    }
  }

  // Strip from process.env immediately. User --require and --import
  // preloads that follow see only empty/absent values.
  delete process.env.MEW_TRANSFORM_ENDPOINT;
  delete process.env.MEW_TRANSFORM_TOKEN;
  delete process.env.MEW_TRANSFORM_OPTIONS;
  delete process.env.MEW_TRANSFORM_OPTS_DIGEST;

  // Export for main-thread consumers (if any).
  module.exports = { endpoint, token, options, optsDigest };
} else {
  // ── Loader context (module.register hooks) ─────────────────────
  // The loader context re-evaluates --require modules before loading
  // the registered hooks. process.env is already stripped by the
  // main-thread run, so we read credentials from the temp file.
  let endpoint = null;
  let token = null;
  let options = '{}';
  let optsDigest = '';

  try {
    if (fs.existsSync(credsFile)) {
      const data = JSON.parse(fs.readFileSync(credsFile, 'utf8'));
      endpoint = data.endpoint || null;
      token = data.token || null;
      options = data.options || '{}';
      optsDigest = data.optsDigest || '';
      // Clean up — credentials are now in module.exports.
      fs.unlinkSync(credsFile);
    }
  } catch (_) {
    // If we can't read the file, leave exports as null.
    // ts-loader will fail with "no endpoint or token" — safe.
  }

  module.exports = { endpoint, token, options, optsDigest };
}
