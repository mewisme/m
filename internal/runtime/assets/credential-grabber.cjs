// Mew credential grabber — runs before any user preload.
// Node processes --require from left to right. Mew places this
// grabber first so it captures transform credentials from process.env,
// strips them before user code executes, and registers the TypeScript
// loader with credentials via module.register()'s data option.
//
// No credentials are written to the filesystem. The handoff uses
// Node's built-in loader registration API, passing data from the
// main thread directly to the loader thread's initialize hook.
// User --require preloads that follow see clean process.env and
// cannot recover credentials.
//
// This module runs twice in Node's two-phase startup:
//   1. Main thread (isMainThread=true): captures env, strips env,
//      registers loader with credentials via module.register().
//      No temp file, no module.exports exposure of real values.
//   2. Loader context (isMainThread=false): re-evaluated by Node's
//      loader thread; exports null values. The loader already
//      received credentials via the initialize hook.
'use strict';

const { isMainThread } = require('node:worker_threads');

if (isMainThread) {
  // ── Main thread ────────────────────────────────────────────────
  // Capture credentials from process.env and strip immediately.
  // credential-grabber runs FIRST (leftmost --require), so no user
  // code has executed yet.
  const endpoint = process.env.MEW_TRANSFORM_ENDPOINT || null;
  const token = process.env.MEW_TRANSFORM_TOKEN || null;
  const options = process.env.MEW_TRANSFORM_OPTIONS || '{}';
  const optsDigest = process.env.MEW_TRANSFORM_OPTS_DIGEST || '';
  const configDir = process.env.MEW_TRANSFORM_CONFIG_DIR || '';

  // Strip from process.env immediately — before any user --require.
  delete process.env.MEW_TRANSFORM_ENDPOINT;
  delete process.env.MEW_TRANSFORM_TOKEN;
  delete process.env.MEW_TRANSFORM_OPTIONS;
  delete process.env.MEW_TRANSFORM_OPTS_DIGEST;
  delete process.env.MEW_TRANSFORM_CONFIG_DIR;

  // Register the TypeScript loader with credentials passed via
  // module.register()'s data option. This is the sole secure
  // handoff path: data travels from this closure directly to the
  // loader thread's initialize hook, never touching the filesystem
  // or module.exports.
  if (endpoint && token) {
    try {
      const { register } = require('node:module');
      const { pathToFileURL } = require('node:url');
      const path = require('node:path');
      const tsLoader = pathToFileURL(path.join(__dirname, 'ts-loader.mjs')).href;
      const parentURL = pathToFileURL(__filename).href;
      register(tsLoader, parentURL, {
        parentURL,
        data: { endpoint, token, options, optsDigest, configDir },
        transferList: [],
      });
    } catch (_) {
      // require('node:module').register not available (Node < 20.6).
      // Fall back to dynamic import. The closure protects credentials;
      // the async call completes before ESM loader initialization.
      import('node:module').then(function (mod) {
        try {
          const { pathToFileURL } = require('node:url');
          const path = require('node:path');
          const tsLoader = pathToFileURL(path.join(__dirname, 'ts-loader.mjs')).href;
          const parentURL = pathToFileURL(__filename).href;
          mod.register(tsLoader, parentURL, {
            parentURL,
            data: { endpoint, token, options, optsDigest, configDir },
            transferList: [],
          });
        } catch (_) { /* registration unavailable */ }
      }).catch(function () { /* import failed — registration unavailable */ });
    }
  }

  // Export null values. Real credentials are never in module.exports.
  module.exports = { endpoint: null, token: null, options: '{}', optsDigest: '', configDir: '' };
} else {
  // ── Loader context ──────────────────────────────────────────────
  // The loader thread re-evaluates --require modules. Credentials
  // were already delivered via the initialize hook; export nulls.
  module.exports = { endpoint: null, token: null, options: '{}', optsDigest: '', configDir: '' };
}
