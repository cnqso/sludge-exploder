package shared

import (
	"os"
	"path/filepath"
)

// appDir returns (creating if needed) the per-user directory all of the
// app/daemon/nmhost's local state lives under. os.UserConfigDir() already
// resolves correctly per-OS (~/Library/Application Support on macOS,
// %AppData% on Windows), so this stays OS-agnostic.
func appDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "SludgeExploder")
	if err := os.MkdirAll(path, 0o700); err != nil {
		return "", err
	}
	return path, nil
}

// HeartbeatSocketPath and ControlSocketPath are declared here but
// implemented per-OS (paths_darwin.go: filesystem socket paths under
// appDir(); paths_windows.go: named-pipe names, which aren't filesystem
// objects and don't live under appDir() at all) -- see shared/ipc.go for
// the matching Listen/Dial seam.

// ControlTokenPath is where the daemon persists the random token used to
// authenticate control-channel connections. 0600, readable only by the
// local user both the app and daemon run as.
func ControlTokenPath() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "control.token"), nil
}

// LockStatePath is where the daemon persists the lock state machine so it
// survives being killed and relaunched (including across a reboot, if
// relaunched afterward).
func LockStatePath() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lock_state.json"), nil
}
