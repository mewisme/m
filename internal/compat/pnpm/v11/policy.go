package v11

// Major is the supported pnpm producer major for v11-shaped locks.
const Major = 11

// LockfileVersion is the on-disk lockfileVersion for pnpm 11 locks.
const LockfileVersion = "9.0"

// BuildPolicyField is the v11 build-policy field name on package entries.
const BuildPolicyField = "buildPolicy"

// OnlyBuiltDependenciesSetting is a v11 settings key for build policy.
const OnlyBuiltDependenciesSetting = "onlyBuiltDependencies"

// IgnoredBuiltDependenciesSetting is a v11 settings key for build policy.
const IgnoredBuiltDependenciesSetting = "ignoredBuiltDependencies"

// ConformanceTarget is the pinned pnpm 11 line for CI.
const ConformanceTarget = "11.17.0"
