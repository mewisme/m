// Mew Web Storage implementation — canonical backend for localStorage
// and sessionStorage.  CJS module required by both preload.cjs (via
// require) and preload.mjs (via createRequire).
//
// localStorage: persisted to disk under the path given by the
//   MEW_LOCAL_STORAGE_PATH env var.  Atomic write (tmp+fsync+rename)
//   on every mutation.  Schema-versioned JSON.  Corrupt files are
//   reset to empty (with a console.warn).
// sessionStorage: in-memory Map, never persisted.
//
// Keys and values are coerced to String.  Missing keys return null.
// Keys are enumerated in insertion order.
//
// Quota: 5 MiB default; override with MEW_STORAGE_QUOTA_BYTES.
// Property-style access (storage.foo, storage[key]) and
// Object.keys(storage) are deliberately unsupported — use the
// Storage methods.

'use strict';

const fs = require('node:fs');
const path = require('node:path');

// ---- constants ---------------------------------------------------------

var SCHEMA_VERSION = 1;
var DEFAULT_QUOTA = 5 * 1024 * 1024; // 5 MiB

// ---- helpers -----------------------------------------------------------

function quotaFromEnv() {
  var raw = process.env.MEW_STORAGE_QUOTA_BYTES;
  if (raw === undefined) return DEFAULT_QUOTA;
  var n = Number(raw);
  if (!Number.isFinite(n) || n <= 0) return DEFAULT_QUOTA;
  return n;
}

function storageError(name, message) {
  // Use DOMException when available (Node 17+), fall back to Error.
  try {
    return new DOMException(message, name);
  } catch (_) {
    var e = new Error(message);
    e.name = name;
    e.code = name; // QuotaExceededError
    return e;
  }
}

// ---- persistence -------------------------------------------------------

// loadStore reads and validates the on-disk JSON store.
// Returns {items, order} or null (missing / empty / corrupt).
function loadStore(filePath) {
  var raw;
  try {
    raw = fs.readFileSync(filePath, 'utf8');
  } catch (e) {
    if (e.code === 'ENOENT') return null;
    throw e;
  }

  var data;
  try {
    data = JSON.parse(raw);
  } catch (_) {
    console.warn('mew: localStorage file corrupt (invalid JSON), resetting.');
    return null;
  }

  if (!data || typeof data !== 'object') {
    console.warn('mew: localStorage file corrupt (not an object), resetting.');
    return null;
  }
  if (data.schemaVersion !== SCHEMA_VERSION) {
    // Unknown schema version — reset rather than misinterpreting.
    console.warn(
      'mew: localStorage schema version ' + data.schemaVersion +
      ' unsupported (expected ' + SCHEMA_VERSION + '), resetting.'
    );
    return null;
  }
  if (!data.items || typeof data.items !== 'object') {
    console.warn('mew: localStorage file corrupt (missing items), resetting.');
    return null;
  }
  if (!Array.isArray(data.order)) {
    console.warn('mew: localStorage file corrupt (missing order), resetting.');
    return null;
  }

  // Rebuild order, filtering out keys whose values aren't strings.
  var order = [];
  var seen = {};
  for (var i = 0; i < data.order.length; i++) {
    var k = data.order[i];
    if (typeof k !== 'string') {
      console.warn('mew: localStorage file corrupt (non-string key in order), resetting.');
      return null;
    }
    if (!(k in data.items)) {
      // Order references a missing key — skip it (graceful).
      continue;
    }
    if (typeof data.items[k] !== 'string') {
      // Value is not a string — skip the key (graceful).
      continue;
    }
    if (seen[k]) continue; // duplicate in order — skip
    seen[k] = true;
    order.push(k);
  }

  // Copy only valid string entries.
  var items = {};
  var itemKeys = Object.keys(data.items);
  for (var j = 0; j < itemKeys.length; j++) {
    var ik = itemKeys[j];
    if (typeof data.items[ik] === 'string') {
      items[ik] = data.items[ik];
    }
  }

  return { items: items, order: order };
}

// saveStore writes the store atomically.
// Sequence: write to unique temp → fsync → rename.
function saveStore(filePath, items, order) {
  var json = JSON.stringify({
    schemaVersion: SCHEMA_VERSION,
    items: items,
    order: order,
  });

  var dir = path.dirname(filePath);
  try {
    fs.mkdirSync(dir, { recursive: true });
  } catch (_) {
    // Directory already exists in race — ignore.
  }

  // Unique temp name to avoid collisions with concurrent writers.
  // pid + timestamp + random suffix.
  var tmpName = filePath + '.tmp.' + process.pid + '.' + Date.now() + '.' +
    Math.random().toString(36).slice(2, 8);
  try {
    fs.writeFileSync(tmpName, json, { flag: 'wx' });
  } catch (e) {
    // If exclusive-write fails (race), try once more with a new name.
    if (e.code === 'EEXIST') {
      tmpName = filePath + '.tmp.' + process.pid + '.' + Date.now() + '.' +
        Math.random().toString(36).slice(2, 8);
      fs.writeFileSync(tmpName, json, { flag: 'wx' });
    } else {
      throw e;
    }
  }

  var fd;
  try {
    fd = fs.openSync(tmpName, 'r+');
    fs.fsyncSync(fd);
  } finally {
    if (fd !== undefined) fs.closeSync(fd);
  }

  // Atomic rename on same filesystem.
  try {
    fs.renameSync(tmpName, filePath);
  } catch (e) {
    // Clean up temp on failure.
    try { fs.unlinkSync(tmpName); } catch (_) { /* best-effort */ }
    throw e;
  }
}

// ---- localStorage ------------------------------------------------------

function createLocalStorage(opts) {
  opts = opts || {};
  var filePath = opts.filePath || null;
  var quota = opts.quota || quotaFromEnv();

  // Mutable state — populated on first access.
  var items = Object.create(null);
  var order = [];
  var totalSize = 0;
  var loaded = false;

  function ensureLoaded() {
    if (loaded) return;
    if (filePath) {
      var stored = loadStore(filePath);
      if (stored) {
        items = stored.items;
        order = stored.order;
        for (var i = 0; i < order.length; i++) {
          totalSize += items[order[i]].length;
        }
      }
    }
    loaded = true;
  }

  function flush() {
    if (!filePath) return;
    saveStore(filePath, items, order);
  }

  function checkQuota(newBytes) {
    if (newBytes > quota) {
      throw storageError(
        'QuotaExceededError',
        "Failed to execute 'setItem' on 'Storage': " +
        'Setting the value exceeded the quota.'
      );
    }
  }

  function computeNewSize(key, newValue) {
    var s = totalSize;
    if (key in items) {
      s -= items[key].length;
    }
    return s + newValue.length;
  }

  return {
    getItem: function (key) {
      ensureLoaded();
      var k = String(key);
      if (!(k in items)) return null;
      return items[k];
    },

    setItem: function (key, value) {
      ensureLoaded();
      var k = String(key);
      var v = String(value);
      var newSize = computeNewSize(k, v);
      checkQuota(newSize);

      if (!(k in items)) {
        order.push(k);
      }
      items[k] = v;
      totalSize = newSize;
      flush();
    },

    removeItem: function (key) {
      ensureLoaded();
      var k = String(key);
      if (!(k in items)) return;

      totalSize -= items[k].length;
      delete items[k];
      var idx = order.indexOf(k);
      if (idx !== -1) order.splice(idx, 1);
      flush();
    },

    clear: function () {
      ensureLoaded();
      items = Object.create(null);
      order = [];
      totalSize = 0;
      flush();
    },

    key: function (index) {
      ensureLoaded();
      if (index < 0 || index >= order.length) return null;
      return order[index];
    },

    get length() {
      ensureLoaded();
      return order.length;
    },
  };
}

// ---- sessionStorage ----------------------------------------------------

function createSessionStorage() {
  var store = new Map();
  return {
    getItem: function (key) {
      var v = store.get(String(key));
      return v === undefined ? null : v;
    },
    setItem: function (key, value) { store.set(String(key), String(value)); },
    removeItem: function (key) { store.delete(String(key)); },
    clear: function () { store.clear(); },
    key: function (index) {
      var keys = Array.from(store.keys());
      return index >= 0 && index < keys.length ? keys[index] : null;
    },
    get length() { return store.size; },
  };
}

// ---- exports -----------------------------------------------------------

module.exports = { createLocalStorage: createLocalStorage, createSessionStorage: createSessionStorage };
