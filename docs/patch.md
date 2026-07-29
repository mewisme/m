# Patch application

Mew applies pnpm patch files to extracted package trees during install.

## Dependencies

[`github.com/bluekeyes/go-gitdiff`](https://github.com/bluekeyes/go-gitdiff) v0.8.1 is a direct module dependency. It parses unified diff input and applies hunks to in-memory file content. Mew wraps it in `internal/archive` for path resolution, sandboxing, and I/O.
