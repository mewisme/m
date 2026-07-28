package v10

// Major is the supported pnpm producer major for v10-shaped locks.
const Major = 10

// LockfileVersion is the on-disk lockfileVersion for pnpm 10 locks.
const LockfileVersion = "9.0"

// PackageChecksumField is the v10+ package-level checksum field name.
const PackageChecksumField = "checksum"

// PatchedDependenciesField is a root-level field observed in pnpm 10 locks.
const PatchedDependenciesField = "patchedDependencies"

// ConfigDependenciesField is a root-level field observed in pnpm 10 locks.
const ConfigDependenciesField = "configDependencies"

// ConformanceTarget is the pinned pnpm 10 line for CI.
const ConformanceTarget = "10.34.5"
