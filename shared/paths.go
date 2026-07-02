package shared

import (
	"os"
	"path/filepath"
)

// SocketPath returns the local Unix-socket path the app listens on and
// nmhost connects to. Windows named-pipe swap is deferred to Stage 5.
func SocketPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(dir, "SludgeExploder")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "app.sock"), nil
}
