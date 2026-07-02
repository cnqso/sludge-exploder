package shared

import (
	"os"
	"path/filepath"
)

// appDir returns (creating if needed) the per-user directory all of the
// app/daemon/nmhost's local state lives under.
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

// HeartbeatSocketPath is the daemon's Unix socket that nmhost relays the
// extension's Native Messaging traffic to (extension <-> daemon).
// Cross-platform named-pipe swap is deferred to Stage 5 Windows parity.
func HeartbeatSocketPath() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "heartbeat.sock"), nil
}

// ControlSocketPath is the daemon's Unix socket that the app dials directly
// (app <-> daemon) to send authenticated control commands.
func ControlSocketPath() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "control.sock"), nil
}

// ControlTokenPath is where the daemon persists the random token used to
// authenticate control-socket connections. 0600, readable only by the local
// user both the app and daemon run as.
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
