// Mew runtime preload (ESM).
// Runs before every user module (--import).
//
// Transform credentials (MEW_TRANSFORM_*) are already stripped by
// credential-grabber.cjs, which runs as the first --require.
// The deletions below are defense-in-depth: in the unlikely event
// credential-grabber did not run, user code still sees clean env.
delete process.env.MEW_TRANSFORM_ENDPOINT;
delete process.env.MEW_TRANSFORM_TOKEN;
delete process.env.MEW_TRANSFORM_OPTIONS;
delete process.env.MEW_TRANSFORM_OPTS_DIGEST;
delete process.env.MEW_TRANSFORM_CONFIG_DIR;

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
