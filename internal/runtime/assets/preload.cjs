// Mew runtime preload (CommonJS).
// Runs before every user module (--require).
// Must NOT strip transform credentials here — this runs before
// loader-register.mjs creates the loader thread, and the loader
// thread needs MEW_TRANSFORM_* in its process.env copy to capture
// them. Credential stripping happens in preload.mjs (--import),
// which runs after loader thread creation.
'use strict';
