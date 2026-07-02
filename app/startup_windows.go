package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// startupRunKeyName is the value name under the per-user Run key -- not the
// same as shared.NativeHostName, just a readable identifier for this entry
// among whatever else is in Run.
const startupRunKeyName = "SludgeExploderDaemon"

const startupRunKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// EnableStartOnLogin registers the daemon to launch automatically at login
// via the per-user Run registry key -- no elevation needed, and trivially
// removable by hand (deleting the value, or via Task Manager's Startup
// tab), matching the "not privileged, always removable" posture carried
// over from Stage 3. This is the Windows equivalent of what a macOS
// LaunchAgent would do (not built yet -- see the Stage 4 plan).
func (a *App) EnableStartOnLogin() error {
	daemonExe, err := daemonPath()
	if err != nil {
		return fmt.Errorf("locating daemon binary: %w", err)
	}
	if _, err := os.Stat(daemonExe); err != nil {
		return fmt.Errorf("daemon binary not found at %s -- run `make build` first", daemonExe)
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, startupRunKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening registry key: %w", err)
	}
	defer key.Close()
	return key.SetStringValue(startupRunKeyName, daemonExe)
}

// DisableStartOnLogin removes the Run key entry. Not an error if it was
// never set.
func (a *App) DisableStartOnLogin() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRunKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening registry key: %w", err)
	}
	defer key.Close()
	if err := key.DeleteValue(startupRunKeyName); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

// IsStartOnLoginEnabled reports whether the Run key entry currently exists.
func (a *App) IsStartOnLoginEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRunKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_, _, err = key.GetStringValue(startupRunKeyName)
	return err == nil
}

// daemonPath returns the absolute path to the daemon binary, expected next
// to the running app binary -- same convention as nmhostPath() in
// hostinstall.go.
func daemonPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "daemon.exe"), nil
}
