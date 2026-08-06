// Mew runtime — loader registration preload.
// Registration moved to credential-grabber.cjs (runs as first --require).
// credential-grabber captures env vars, strips them, and calls
// module.register() with credential data via the data option.
// This file is kept as a non-injected loader-support asset in the
// manifest; it is no longer injected into Node argv.
