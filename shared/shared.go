// Package shared holds types and constants used by both /app and /daemon:
// IPC message shapes, the config schema, and protocol constants. It has no
// behavior of its own yet — populated as Stage 2/3 wire the IPC and
// heartbeat protocols described in docs/ENFORCEMENT.md §5.
package shared

// Version is the current protocol/schema version, bumped when the shapes
// below change in an incompatible way.
const Version = "0.0.0"
