// Mew TypeScript loader — transform hook.
// Communicates with the Go transform service over local TCP.
import { connect } from 'node:net';

let endpoint = null;
let token = null;
let conn = null;
let seq = 0;
let pendingResolve = null;

function ensureEnv() {
  if (endpoint && token) return true;
  endpoint = process.env.MEW_TRANSFORM_ENDPOINT;
  token = process.env.MEW_TRANSFORM_TOKEN;
  return !!(endpoint && token);
}

export function stripPrivateEnv() {
  delete process.env.MEW_TRANSFORM_ENDPOINT;
  delete process.env.MEW_TRANSFORM_TOKEN;
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
            pendingResolve = null;
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
            if (pendingResolve) {
              const r = pendingResolve;
              pendingResolve = null;
              r(JSON.parse(body));
            }
          } catch (e) { /* drop malformed frame */ }
        }
      }

      c.on('data', onData);
      c.once('error', (e) => { conn = null; reject(e); });
      c.once('close', () => { conn = null; });
    });
    c.once('error', (e) => { conn = null; reject(e); });
  });
}

function sendFrame(c, obj) {
  return new Promise((resolve) => {
    pendingResolve = resolve;
    const body = Buffer.from(JSON.stringify(obj), 'utf8');
    const header = Buffer.alloc(4);
    header.writeUInt32LE(body.length, 0);
    c.write(Buffer.concat([header, body]));
  });
}

async function sendTransform(path, source) {
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
    options: '{}',
    opts_digest: '',
    node_major: process.versions.node ? parseInt(process.versions.node.split('.')[0], 10) : 20,
    source_map: 'inline',
  });
  if (!resp || !resp.ok) {
    throw new Error(resp?.error || 'transform failed');
  }
  return resp.code;
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
