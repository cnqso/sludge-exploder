package shared

import "path/filepath"

// HeartbeatSocketPath is the daemon's Unix socket that nmhost relays the
// extension's Native Messaging traffic to (extension <-> daemon).
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
