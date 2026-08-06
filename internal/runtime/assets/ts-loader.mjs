// Mew TypeScript loader — transform hook.
// Communicates with the Go transform service over local TCP.
import { connect } from 'node:net';
import { createHash } from 'node:crypto';

// Capture credentials at module load time and strip from process.env
// immediately, before any user module executes. Credentials are held in
// closure variables reachable only by this loader, never by user code.
const endpoint = process.env.MEW_TRANSFORM_ENDPOINT || null;
const token = process.env.MEW_TRANSFORM_TOKEN || null;
const transformOptions = process.env.MEW_TRANSFORM_OPTIONS || '{}';

delete process.env.MEW_TRANSFORM_ENDPOINT;
delete process.env.MEW_TRANSFORM_TOKEN;
delete process.env.MEW_TRANSFORM_OPTIONS;
delete process.env.MEW_TRANSFORM_OPTS_DIGEST;

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

async function sendTransform(path, source) {
  let lastErr;
  const sourceStr = String(source);
  const sourceDigest = createHash('sha256').update(sourceStr).digest('hex');
  const optsDigest = transformOptions ? createHash('sha256').update(transformOptions).digest('hex') : '';

  for (let attempt = 0; attempt < 2; attempt++) {
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
  return 'ts';
}

function formatFromPath(p) {
  if (p.endsWith('.cts')) return 'cjs';
  return 'esm';
}

// ── Node loader hooks ──────────────────────────────────────────────

export function stripPrivateEnv() {
  // Credentials were stripped from process.env at module load time.
  // This function exists for explicit cleanup if needed; it is a no-op
  // for the env side because process.env is already clean.
}

// resolve hook: mark TypeScript files.
export async function resolve(specifier, context, nextResolve) {
  if (!specifier) return nextResolve(specifier, context);
  try {
    const url = new URL(specifier, context.parentURL || 'file:///');
    if (url.protocol !== 'file:') return nextResolve(specifier, context);
    const path = decodeURIComponent(url.pathname);
    if (path.endsWith('.ts') || path.endsWith('.mts') || path.endsWith('.cts')) {
      return {
        format: path.endsWith('.cts') ? 'commonjs' : 'module',
        url: url.href,
        shortCircuit: true,
      };
    }
  } catch (_) { /* not a valid URL, delegate */ }
  return nextResolve(specifier, context);
}

// load hook: transform TypeScript to JavaScript.
export async function load(url, context, nextLoad) {
  if (context.format !== 'module' && context.format !== 'commonjs') {
    return nextLoad(url, context);
  }
  const pathname = decodeURIComponent(new URL(url).pathname);
  if (!pathname.endsWith('.ts') && !pathname.endsWith('.mts') && !pathname.endsWith('.cts')) {
    return nextLoad(url, context);
  }
  const source = await nextLoad(url, { ...context, format: context.format });
  const code = await sendTransform(pathname, String(source.source));
  stripPrivateEnv();
  return { format: context.format, source: code, shortCircuit: true };
}
