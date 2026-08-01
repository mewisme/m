// Mew runtime — loader registration preload.
// Registers the TypeScript loader hook via module.register()
// before any user code executes.

// Load MEW_TRANSFORM_ENDPOINT and MEW_TRANSFORM_TOKEN from env
// (set by Go parent). These are consumed by ts-loader.mjs.
// The loader will strip them before user code sees process.env.

import { register } from 'node:module';

const tsLoader = new URL('./ts-loader.mjs', import.meta.url).href;

// Register our loader hooks. Node calls these for every module load.
register(tsLoader, import.meta.url, {
  parentURL: import.meta.url,
  data: {},
  transferList: [],
});
