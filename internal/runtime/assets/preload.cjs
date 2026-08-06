// Mew runtime preload (CommonJS).
// Runs before every user module (--require).
// Must NOT strip transform credentials here — this runs before
// loader-register.mjs creates the loader thread, and the loader
// thread needs MEW_TRANSFORM_* in its process.env copy to capture
// them. Credential stripping happens in preload.mjs (--import),
// which runs after loader thread creation.
'use strict';

// Web Storage polyfill (localStorage, sessionStorage).
// ponytail: in-memory only; localStorage disk persistence deferred to 0055+.
{
  const createStorage = () => {
    const store = new Map();
    return {
      getItem(key) {
        const v = store.get(String(key));
        return v === undefined ? null : v;
      },
      setItem(key, value) { store.set(String(key), String(value)); },
      removeItem(key) { store.delete(String(key)); },
      clear() { store.clear(); },
      key(index) {
        const keys = [...store.keys()];
        return index >= 0 && index < keys.length ? keys[index] : null;
      },
      get length() { return store.size; },
    };
  };
  globalThis.localStorage = createStorage();
  globalThis.sessionStorage = createStorage();
}
