// Mew runtime preload (ESM).
// Runs before every user module (--import). This runs AFTER
// loader-register.mjs, which creates the loader thread. The loader
// thread already captured MEW_TRANSFORM_* into its process.env copy
// at creation time. We strip the main-thread copy now so user code
// can never read these credentials.
delete process.env.MEW_TRANSFORM_ENDPOINT;
delete process.env.MEW_TRANSFORM_TOKEN;
delete process.env.MEW_TRANSFORM_OPTIONS;
delete process.env.MEW_TRANSFORM_OPTS_DIGEST;
