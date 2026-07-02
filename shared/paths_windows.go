package shared

// HeartbeatSocketPath and ControlSocketPath return named-pipe names on
// Windows -- these aren't filesystem paths (no appDir(), no directory to
// create; the pipe namespace is separate from the filesystem), unlike
// their macOS Unix-socket counterparts in paths_darwin.go.
func HeartbeatSocketPath() (string, error) {
	return `\\.\pipe\SludgeExploderHeartbeat`, nil
}

func ControlSocketPath() (string, error) {
	return `\\.\pipe\SludgeExploderControl`, nil
}
