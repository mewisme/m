# Cross-device copy fallback fixture (optional manual CI)

When the store root and project `node_modules` are on different volumes, the
link planner must select `OpCopy`. Use this marker in manual CI on APFS/btrfs
setups; automated tests mock cross-device via `planner.Capabilities{SameVolume:false}`.
