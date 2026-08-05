// Mew TypeScript loader — transform hook.
// Communicates with the Go transform service over local TCP.
import { connect } from 'node:net';

// Capture credentials at module load time and strip from process.env
// immediately, before any user module executes. Credentials are held in
// closure variables reachable only by this loader, never by user code.
const endpoint = process.env.MEW_TRANSFORM_ENDPOINT || null;
const token = process.env.MEW_TRANSFORM_TOKEN || null;
const transformOptions = process.env.MEW_TRANSFORM_OPTIONS || '{}';
const optsDigest = process.env.MEW_TRANSFORM_OPTS_DIGEST || '';

delete process.env.MEW_TRANSFORM_ENDPOINT;
delete process.env.MEW_TRANSFORM_TOKEN;
delete process.env.MEW_TRANSFORM_OPTIONS;
delete process.env.MEW_TRANSFORM_OPTS_DIGEST;

let conn = null;
let seq = 0;
// Per-request resolve/reject map keyed by request ID for concurrent transforms.
const pending = new Map();

const DEFAULT_TIMEOUT = 60000; // 60s per request

function ensureEnv() {
  return !!(endpoint && token);
}

export function stripPrivateEnv() {
  // Credentials were stripped from process.env at module load time.
  // This function exists for explicit cleanup if needed; it is a no-op
  // for the env side because process.env is already clean.
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
  return new Promise((resolve, reject) => {
    if (conn && !conn.destroyed) { resolve(conn); return; }
    if (!ensureEnv()) { reject(new Error('no endpoint or token')); return; }
    const [host, portStr] = endpoint.split(':');
    const port = parseInt(portStr, 10);
    let buf = Buffer.alloc(0);
    let headerLen = -1;

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
            if (!msg.ok) { reject(new Error(msg.reason || 'auth failed')); return; }
            // Auth succeeded; switch to request-response mode.
            c.removeAllListeners('data');
            c.on('data', onResponse);
            resolve(c);
          } catch (e) { reject(e); }
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
                entry.reject(new Error(resp.error || 'transform failed'));
              }
            }
            // Unknown/duplicate IDs: drop silently (timed-out or already resolved).
          } catch (e) { /* drop malformed frame */ }
        }
      }

      c.on('data', onData);
      c.once('error', (e) => { conn = null; rejectAllPending(e); reject(e); });
      c.once('close', () => {
        conn = null;
        rejectAllPending(new Error('transform service disconnected'));
      });
    });
    c.once('error', (e) => { conn = null; rejectAllPending(e); reject(e); });
  });
}

function sendFrame(c, obj, timeoutMs) {
  return new Promise((resolve, reject) => {
    const id = obj.id;
    const timer = setTimeout(() => {
      if (pending.has(id)) {
        pending.delete(id);
        reject(new Error(`transform request ${id} timed out`));
      }
    }, timeoutMs || DEFAULT_TIMEOUT);
    pending.set(id, { resolve, reject, timer });
    const body = Buffer.from(JSON.stringify(obj), 'utf8');
    const header = Buffer.alloc(4);
    header.writeUInt32LE(body.length, 0);
    c.write(Buffer.concat([header, body]));
  });
}

async function sendTransform(path, source) {
  let lastErr;
  for (let attempt = 0; attempt < 2; attempt++) {
    try {
      const c = await getConn();
      const id = String(++seq);
      const resp = await sendFrame(c, {
        v: 2,
        id,
        op: 'transform',
        path,
        source: String(source),
        source_digest: '',
        loader: loaderFromPath(path),
        format: formatFromPath(path),
        options: transformOptions,
        opts_digest: optsDigest,
        node_major: process.versions.node ? parseInt(process.versions.node.split('.')[0], 10) : 20,
        source_map: 'inline',
        cancel_token: id,
      });
      if (!resp || !resp.ok) {
        const errCode = resp?.err_code || '';
        // No retry for syntax errors or policy failures.
        if (errCode.startsWith('ERR_M_TRANSFORM_SYNTAX') ||
            errCode.startsWith('ERR_M_TRANSFORM_CONFIG') ||
            errCode.startsWith('ERR_M_TRANSFORM_PROTOCOL')) {
          throw new Error(resp?.error || 'transform failed');
        }
        throw new Error(resp?.error || 'transform failed');
      }
      return resp.code;
    } catch (e) {
      lastErr = e;
      if (attempt === 0) {
        // Reset connection on error so next attempt reconnects.
        conn = null;
      }
    }
  }
  throw lastErr;
}

function loaderFromPath(p) {
  if (p.endsWith('.mts')) return 'mts';
  if (p.endsWith('.cts')) return 'cts';
  return 'ts';
}

function formatFromPath(p) {
  if (p.endsWith('.cts')) return 'cjs';
  return 'esm';
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
