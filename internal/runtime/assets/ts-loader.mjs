// Mew TypeScript loader — transform hook.
// Communicates with the Go transform service over local TCP.
import { connect } from 'node:net';
import { createHash } from 'node:crypto';
import { accessSync } from 'node:fs';
import { resolve as pathResolve, parse as pathParse, join as pathJoin, dirname } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

// Credentials are received via the initialize hook, passed from
// credential-grabber.cjs through module.register()'s data option.
// No filesystem artifact, no env var, no module cache snooping.
let endpoint = null;
let token = null;
let transformOptions = '{}';
let optsDigest = '';
let configDir = '';

export function initialize(data) {
  if (data && data.endpoint && data.token) {
    endpoint = data.endpoint;
    token = data.token;
    transformOptions = data.options || '{}';
    optsDigest = data.optsDigest || '';
    configDir = data.configDir || '';
  }
}

let conn = null;
let seq = 0;
// Per-request resolve/reject map keyed by request ID for concurrent transforms.
const pending = new Map();

// Reconnect guard: non-null while a reconnect attempt is in progress.
// Concurrent callers await the same promise instead of starting their own.
let reconnectPromise = null;

const DEFAULT_TIMEOUT = 60000; // 60s per request

// ── Typed transform error ──────────────────────────────────────────

// Non-retryable error code prefixes. Any err_code starting with one of these
// (or matching exactly) must not be retried.
const NON_RETRYABLE_PREFIXES = [
  'ERR_M_TRANSFORM_SYNTAX',
  'ERR_M_TRANSFORM_CONFIG',
  'ERR_M_TRANSFORM_PROTOCOL',
  'ERR_M_TRANSFORM_AUTH',
  'ERR_M_TRANSFORM_FRAME_SIZE',
  'ERR_M_TRANSFORM_TIMEOUT',
  'ERR_M_TRANSFORM_CANCELLED',
  'ERR_M_TRANSFORM_UNAVAILABLE',
  'ERR_M_TRANSFORM_CACHE_CORRUPT',
  'ERR_M_TRANSFORM_ENGINE',
  'ERR_M_TRANSFORM_UNSUPPORTED',
  'ERR_M_POLICY',
  'ERR_M_INTEGRITY',
  'ERR_M_UNSUPPORTED',
  'ERR_M_USAGE',
  'ERR_M_CANCELLED',
];

// Transient transport error substrings — retryable exactly once.
const TRANSIENT_SUBSTRINGS = [
  'ECONNRESET',
  'ECONNREFUSED',
  'EPIPE',
  'connection reset',
  'connection refused',
  'broken pipe',
  'transform service disconnected',
];

class TransformError extends Error {
  constructor(errCode, message, requestId, retryable) {
    super(message);
    this.name = 'TransformError';
    this.errCode = errCode;
    this.requestId = requestId;
    this.retryable = retryable;
  }
}

function classifyError(err, requestId) {
  // Already a TransformError — preserve its classification.
  if (err instanceof TransformError) return err;

  const msg = err?.message || '';
  const code = err?.code || '';

  // Node.js system errors with code.
  if (code) {
    const transient = TRANSIENT_SUBSTRINGS.some(s => code.includes(s) || msg.includes(s));
    return new TransformError('', msg, requestId, transient);
  }

  // Bare message — check for transient substrings.
  const transient = TRANSIENT_SUBSTRINGS.some(s => msg.includes(s));
  return new TransformError('', msg, requestId, transient);
}

function isNonRetryableErrCode(errCode) {
  if (!errCode) return false;
  return NON_RETRYABLE_PREFIXES.some(p => errCode.startsWith(p));
}

// ── Connection management ──────────────────────────────────────────

function ensureEnv() {
  return !!(endpoint && token);
}

// Reject all pending requests — called on disconnect or fatal error.
function rejectAllPending(err) {
  for (const [id, entry] of pending) {
    clearTimeout(entry.timer);
    entry.reject(err);
  }
  pending.clear();
}

function getConn() {
  // If a reconnect is already in progress, share it.
  if (reconnectPromise) return reconnectPromise;

  // Existing healthy connection: reuse.
  if (conn && !conn.destroyed) {
    return Promise.resolve(conn);
  }

  if (!ensureEnv()) {
    return Promise.reject(new TransformError('', 'no endpoint or token', '', false));
  }

  reconnectPromise = new Promise((resolve, reject) => {
    const [host, portStr] = endpoint.split(':');
    const port = parseInt(portStr, 10);
    let buf = Buffer.alloc(0);
    let headerLen = -1;
    let resolved = false;

    function settle(err, c) {
      if (resolved) return;
      resolved = true;
      reconnectPromise = null;
      if (err) reject(err);
      else resolve(c);
    }

    const c = connect({ host, port }, () => {
      conn = c;
      // Authenticate immediately.
      const authBody = Buffer.from(JSON.stringify({ v: 2, token }), 'utf8');
      const authHdr = Buffer.alloc(4);
      authHdr.writeUInt32LE(authBody.length, 0);
      c.write(Buffer.concat([authHdr, authBody]));

      function onData(chunk) {
        buf = Buffer.concat([buf, chunk]);
        while (true) {
          if (headerLen < 0) {
            if (buf.length < 4) return;
            headerLen = buf.readUInt32LE(0);
            buf = buf.subarray(4);
          }
          if (buf.length < headerLen) return;
          const body = buf.subarray(0, headerLen).toString('utf8');
          buf = buf.subarray(headerLen);
          headerLen = -1;
          try {
            const msg = JSON.parse(body);
            if (!msg.ok) {
              settle(new TransformError(msg.err_code || 'ERR_M_TRANSFORM_AUTH', msg.reason || 'auth failed', '', false), null);
              return;
            }
            // Auth succeeded; switch to request-response mode.
            c.removeAllListeners('data');
            c.on('data', onResponse);
            settle(null, c);
          } catch (e) { settle(e, null); }
          return;
        }
      }

      function onResponse(chunk) {
        buf = Buffer.concat([buf, chunk]);
        while (true) {
          if (headerLen < 0) {
            if (buf.length < 4) return;
            headerLen = buf.readUInt32LE(0);
            buf = buf.subarray(4);
          }
          if (buf.length < headerLen) return;
          const body = buf.subarray(0, headerLen).toString('utf8');
          buf = buf.subarray(headerLen);
          headerLen = -1;
          try {
            const resp = JSON.parse(body);
            const id = resp.id;
            if (id != null && pending.has(id)) {
              const entry = pending.get(id);
              clearTimeout(entry.timer);
              pending.delete(id);
              if (resp.ok) {
                entry.resolve(resp);
              } else {
                const errCode = resp.err_code || '';
                entry.reject(new TransformError(
                  errCode,
                  resp.error || 'transform failed',
                  id,
                  false, // non-OK responses are never retryable
                ));
              }
            } else if (id != null) {
              // ID parsed but no pending entry: response arrived after timeout
              // or cancellation. Drop silently.
            } else {
              // Response with no ID — malformed. Fail the oldest pending
              // request if any exist, so it doesn't wait for timeout.
              if (pending.size > 0) {
                const oldestKey = pending.keys().next().value;
                const oldest = pending.get(oldestKey);
                if (oldest) {
                  clearTimeout(oldest.timer);
                  pending.delete(oldestKey);
                  oldest.reject(new TransformError(
                    '',
                    'malformed response from transform service',
                    oldestKey,
                    false,
                  ));
                }
              }
            }
            // Unknown/duplicate IDs: drop silently (timed-out or already resolved).
          } catch (e) { /* drop malformed frame */ }
        }
      }

      c.on('data', onData);
      c.once('error', (e) => {
        conn = null;
        if (!resolved) settle(e, null);
        rejectAllPending(classifyError(e, ''));
      });
      c.once('close', () => {
        conn = null;
        rejectAllPending(new TransformError('', 'transform service disconnected', '', true));
      });
    });
    c.once('error', (e) => {
      conn = null;
      if (!resolved) settle(e, null);
      rejectAllPending(classifyError(e, ''));
    });
  });

  return reconnectPromise;
}

// ── Frame send / cancel ────────────────────────────────────────────

// sendCancel sends an OpCancel frame. It does NOT create a pending entry
// or set a timer — cancel messages are fire-and-forget.
function sendCancel(c, cancelToken) {
  const id = String(++seq);
  const body = Buffer.from(JSON.stringify({
    v: 2,
    id,
    op: 'cancel',
    cancel_token: cancelToken,
  }), 'utf8');
  const header = Buffer.alloc(4);
  header.writeUInt32LE(body.length, 0);
  try {
    c.write(Buffer.concat([header, body]));
  } catch (_) {
    // Best-effort; connection may already be dead.
  }
}

function sendFrame(c, obj, timeoutMs) {
  return new Promise((resolve, reject) => {
    const id = obj.id;
    const cancelToken = obj.cancel_token || id;

    const timer = setTimeout(() => {
      // Atomically remove this request and send cancel.
      if (pending.has(id)) {
        pending.delete(id);
        // Best-effort cancel notification.
        sendCancel(c, cancelToken);
      }
      reject(new TransformError(
        'ERR_M_TRANSFORM_TIMEOUT',
        `transform request ${id} timed out`,
        id,
        false,
      ));
    }, timeoutMs || DEFAULT_TIMEOUT);

    pending.set(id, { resolve, reject, timer });
    const body = Buffer.from(JSON.stringify(obj), 'utf8');
    const header = Buffer.alloc(4);
    header.writeUInt32LE(body.length, 0);
    c.write(Buffer.concat([header, body]));
  });
}

// ── Transform orchestration ────────────────────────────────────────

async function sendTransform(path, source, signal) {
  let lastErr;
  const sourceStr = String(source);
  const sourceDigest = createHash('sha256').update(sourceStr).digest('hex');
  const optsDigest = transformOptions ? createHash('sha256').update(transformOptions).digest('hex') : '';

  for (let attempt = 0; attempt < 2; attempt++) {
    // Abort early if caller has cancelled.
    if (signal?.aborted) {
      throw new TransformError('ERR_M_CANCELLED', 'transform aborted', '', false);
    }

    try {
      const c = await getConn();
      const id = String(++seq);
      const resp = await sendFrame(c, {
        v: 2,
        id,
        op: 'transform',
        path,
        source: sourceStr,
        source_digest: sourceDigest,
        loader: loaderFromPath(path),
        format: formatFromPath(path),
        options: transformOptions,
        opts_digest: optsDigest,
        node_major: process.versions.node ? parseInt(process.versions.node.split('.')[0], 10) : 20,
        source_map: 'inline',
        cancel_token: id,
      });

      // Valid response received — return result.
      return resp.code;
    } catch (e) {
      lastErr = classifyError(e, '');

      // Never retry if the caller has cancelled.
      if (signal?.aborted) break;

      // Never retry if the error is non-retryable.
      if (!lastErr.retryable) break;

      // Reset connection on transient error so next attempt reconnects.
      if (attempt === 0) {
        conn = null;
        continue;
      }
      // Second transient failure: don't retry further.
      break;
    }
  }

  throw lastErr;
}

// ── Path helpers ───────────────────────────────────────────────────

function loaderFromPath(p) {
  if (p.endsWith('.mts')) return 'mts';
  if (p.endsWith('.cts')) return 'cts';
  if (p.endsWith('.tsx')) return 'tsx';
  return 'ts';
}

function formatFromPath(p) {
  if (p.endsWith('.cts')) return 'cjs';
  return 'esm';
}

// ── PnP resolution adapter ─────────────────────────────────────────

let pnpApi = null;
let pnpChecked = false;

function ensurePnp(parentPath) {
  if (pnpChecked) return;
  pnpChecked = true;
  // Walk up from the parent path to find .pnp.cjs.
  let dir = parentPath ? dirname(parentPath) : resolveBaseDir;
  if (!dir) return;
  const root = findProjectRoot(dir);
  if (!root) return;
  try {
    // .pnp.cjs exports the PnP API (resolveRequest, etc.).
    const pnpPath = pathJoin(root, '.pnp.cjs');
    accessSync(pnpPath); // throws if missing
    pnpApi = createRequire(import.meta.url)(pnpPath);
  } catch (_) { /* PnP not available */ }
}

// findProjectRoot walks up from dir looking for .pnp.cjs or .pnp.data.json.
// Returns the directory containing the PnP artifact, or null.
function findProjectRoot(dir) {
  let current = pathResolve(dir);
  const { root } = pathParse(current);
  while (current !== root) {
    try {
      accessSync(pathJoin(current, '.pnp.cjs'));
      return current;
    } catch (_) {}
    try {
      accessSync(pathJoin(current, '.pnp.data.json'));
      return current; // PnP project without .pnp.cjs (unplugged) — detected but no API.
    } catch (_) {}
    const parent = dirname(current);
    if (parent === current) break;
    current = parent;
  }
  return null;
}

// ── Path alias resolution ──────────────────────────────────────────

let resolveBaseDir = '';
let resolvePaths = null;
let pathsParsed = false;

function ensurePathsParsed() {
  if (pathsParsed) return;
  pathsParsed = true;
  try {
    const opts = JSON.parse(transformOptions || '{}');
    if (opts.paths && typeof opts.paths === 'object') {
      resolvePaths = opts.paths;
    }
    resolveBaseDir = configDir || '';
    if (opts.baseUrl && resolveBaseDir) {
      resolveBaseDir = pathResolve(resolveBaseDir, opts.baseUrl);
    }
  } catch (_) { /* parse failure: leave disabled */ }
}

// matchPathPattern returns captured wildcard values, or null on no match.
// TypeScript paths patterns: "@app/*" matches "@app/helpers" → ["helpers"].
function matchPathPattern(specifier, pattern) {
  if (!pattern.includes('*')) {
    if (specifier === pattern) return [''];
    return null;
  }
  const parts = pattern.split('*');
  if (parts.length === 2) {
    const [prefix, suffix] = parts;
    if (specifier.startsWith(prefix) && specifier.endsWith(suffix) &&
        specifier.length >= prefix.length + (suffix ? suffix.length : 0)) {
      const captured = specifier.slice(prefix.length, suffix ? specifier.length - suffix.length : specifier.length);
      return [captured];
    }
    return null;
  }
  // Multiple wildcards: sequential match.
  let remaining = specifier;
  const captures = [];
  for (let i = 0; i < parts.length; i++) {
    const part = parts[i];
    if (i === 0) {
      if (!remaining.startsWith(part)) return null;
      remaining = remaining.slice(part.length);
    } else if (i === parts.length - 1) {
      if (part === '') { captures.push(remaining); break; }
      if (!remaining.endsWith(part)) return null;
      captures.push(remaining.slice(0, remaining.length - part.length));
    } else {
      const idx = remaining.indexOf(part);
      if (idx === -1) return null;
      captures.push(remaining.slice(0, idx));
      remaining = remaining.slice(idx + part.length);
    }
  }
  return captures;
}

const TS_EXT_PROBE = ['.ts', '.tsx', '.mts', '.cts'];
const JS_EXT_PROBE = ['.js', '.mjs', '.cjs'];
const ALL_EXT_PROBE = [...TS_EXT_PROBE, ...JS_EXT_PROBE];

function tryResolveFile(basePath) {
  // Exact path exists.
  try { accessSync(basePath); return basePath; } catch (_) {}
  // Already has a supported extension — don't probe further.
  const parsed = pathParse(basePath);
  if (parsed.ext && (TS_EXT_PROBE.includes(parsed.ext) || JS_EXT_PROBE.includes(parsed.ext))) {
    return null;
  }
  // Probe extensions.
  for (const ext of ALL_EXT_PROBE) {
    const candidate = basePath + ext;
    try { accessSync(candidate); return candidate; } catch (_) {}
  }
  // Strip unsupported extension and re-probe (e.g. import './foo.js' → './foo.ts').
  if (parsed.ext && !TS_EXT_PROBE.includes(parsed.ext) && !JS_EXT_PROBE.includes(parsed.ext)) {
    return null;
  }
  return null;
}

function resolveViaPaths(specifier, parentURL) {
  ensurePathsParsed();
  if (!resolvePaths) return null;
  for (const [pattern, replacements] of Object.entries(resolvePaths)) {
    const captures = matchPathPattern(specifier, pattern);
    if (!captures) continue;
    for (const replacement of replacements) {
      // Substitute captured values into replacement.
      let resolved = replacement;
      for (let i = 0; i < captures.length; i++) {
        resolved = resolved.replace('*', captures[i]);
      }
      // Resolve relative to baseUrl.
      const fullPath = resolveBaseDir ? pathResolve(resolveBaseDir, resolved) : pathResolve(resolved);
      const found = tryResolveFile(fullPath);
      if (found) return found;
    }
  }
  return null;
}

// fileExists checks whether an absolute file path exists on disk.
function fileExists(absPath) {
  try { accessSync(absPath); return true; } catch (_) { return false; }
}

// probeTypeScriptExtension checks whether a .js/.mjs/.cjs resolved path
// has a corresponding .ts/.tsx/.mts/.cts file that should be used instead.
function probeTypeScriptExtension(resolvedPath) {
  const parsed = pathParse(resolvedPath);
  const jsExts = ['.js', '.mjs', '.cjs'];
  if (!jsExts.includes(parsed.ext)) return null;
  const baseName = pathJoin(parsed.dir, parsed.name);
  // Map .js→.ts, .mjs→.mts, .cjs→.cts; also probe .tsx.
  const probeExts = parsed.ext === '.mjs' ? ['.mts'] :
                    parsed.ext === '.cjs' ? ['.cts'] :
                    ['.ts', '.tsx'];
  for (const ext of probeExts) {
    const candidate = baseName + ext;
    try { accessSync(candidate); return candidate; } catch (_) {}
  }
  return null;
}

function formatFromResolvedPath(p) {
  if (p.endsWith('.mjs') || p.endsWith('.mts')) return 'module';
  if (p.endsWith('.cjs') || p.endsWith('.cts')) return 'commonjs';
  if (p.endsWith('.js') || p.endsWith('.ts') || p.endsWith('.tsx')) return 'module';
  return undefined; // let Node decide
}

// ── Node loader hooks ──────────────────────────────────────────────

export function stripPrivateEnv() {
  // Credentials were stripped from process.env at module load time.
  // This function exists for explicit cleanup if needed; it is a no-op
  // for the env side because process.env is already clean.
}

// resolve hook: augment Node resolution with TypeScript paths, extension mapping.
export async function resolve(specifier, context, nextResolve) {
  if (!specifier) return nextResolve(specifier, context);

  // Try Node's native resolution first.
  let resolved;
  try {
    resolved = await nextResolve(specifier, context);
  } catch (_) {
    resolved = null;
  }

  // Case 1: Node resolved successfully.
  if (resolved && resolved.url) {
    try {
      const url = new URL(resolved.url);
      if (url.protocol === 'file:') {
        const absPath = fileURLToPath(resolved.url);
        // If it's already a TypeScript file, mark format and return.
        if (absPath.endsWith('.ts') || absPath.endsWith('.tsx') || absPath.endsWith('.mts') || absPath.endsWith('.cts')) {
          return {
            format: absPath.endsWith('.cts') ? 'commonjs' : 'module',
            url: resolved.url,
            shortCircuit: true,
          };
        }
        // .js→.ts extension mapping: if the resolved .js file doesn't exist,
        // probe for a .ts variant.
        if (!fileExists(absPath)) {
          const tsPath = probeTypeScriptExtension(absPath);
          if (tsPath) {
            return {
              format: formatFromResolvedPath(tsPath),
              url: pathToFileURL(tsPath).href,
              shortCircuit: true,
            };
          }
        }
      }
      // Not a file we handle; pass through Node's result.
      return resolved;
    } catch (_) { /* malformed URL — pass through */ }
    return resolved;
  }

  // Case 2: Node failed. Try tsconfig paths, then extension probing.
  if (specifier.startsWith('.') || specifier.startsWith('/') || (!specifier.includes(':') && !specifier.startsWith('@'))) {
    // Relative, absolute, or bare specifier — try extension probing first.
    const parentPath = context.parentURL ? fileURLToPath(context.parentURL) : '/';
    const parentDir = dirname(parentPath);
    const candidatePath = specifier.startsWith('.')
      ? pathResolve(parentDir, specifier)
      : (specifier.startsWith('/') ? specifier : null);
    if (candidatePath) {
      const found = tryResolveFile(candidatePath);
      if (found) {
        return {
          format: formatFromResolvedPath(found),
          url: pathToFileURL(found).href,
          shortCircuit: true,
        };
      }
    }
  }

  // Try PnP resolution if .pnp.cjs is available.
  ensurePnp(context.parentURL ? fileURLToPath(context.parentURL) : null);
  if (pnpApi && typeof pnpApi.resolveRequest === 'function') {
    try {
      const issuer = context.parentURL || pathToFileURL('/').href;
      const pnpResult = pnpApi.resolveRequest(specifier, issuer);
      if (pnpResult) {
        return {
          format: formatFromResolvedPath(pnpResult),
          url: pathToFileURL(pnpResult).href,
          shortCircuit: true,
        };
      }
    } catch (_) { /* PnP resolution failed; fall through */ }
  }

  // Try tsconfig paths matching.
  const pathsResolved = resolveViaPaths(specifier, context.parentURL);
  if (pathsResolved) {
    return {
      format: formatFromResolvedPath(pathsResolved),
      url: pathToFileURL(pathsResolved).href,
      shortCircuit: true,
    };
  }

  // All augmentations failed — re-throw the original Node error if we caught one,
  // or delegate to nextResolve (which will throw).
  if (resolved === null) {
    return nextResolve(specifier, context);
  }
  return resolved;
}

// load hook: transform TypeScript to JavaScript.
export async function load(url, context, nextLoad) {
  if (context.format !== 'module' && context.format !== 'commonjs') {
    return nextLoad(url, context);
  }
  const pathname = decodeURIComponent(new URL(url).pathname);
  if (!pathname.endsWith('.ts') && !pathname.endsWith('.tsx') && !pathname.endsWith('.mts') && !pathname.endsWith('.cts')) {
    return nextLoad(url, context);
  }
  const source = await nextLoad(url, { ...context, format: context.format });
  const code = await sendTransform(pathname, String(source.source), context.signal);
  stripPrivateEnv();
  return { format: context.format, source: code, shortCircuit: true };
}
