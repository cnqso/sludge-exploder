//go:build !windows

package main

import "fmt"

// Start-on-login is Windows-only for now (a per-user Registry Run key --
// see startup_windows.go). A macOS LaunchAgent equivalent is a natural
// follow-up, not built yet; Stage 3's daemon stays fully manual on macOS
// in the meantime.
func (a *App) EnableStartOnLogin() error {
	return fmt.Errorf("start-on-login isn't implemented on this platform yet")
}

func (a *App) DisableStartOnLogin() error {
	return fmt.Errorf("start-on-login isn't implemented on this platform yet")
}

func (a *App) IsStartOnLoginEnabled() bool {
	return false
}
