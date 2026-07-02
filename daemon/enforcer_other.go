//go:build !darwin && !windows

package main

import "fmt"

// Linux parity is a future stage (see enforcer_darwin.go and
// enforcer_windows.go for the two backends built so far, and
// docs/ENFORCEMENT.md §4.3's "one daemon, N backends" plan).
type unsupportedEnforcer struct{}

func newEnforcer() Enforcer {
	return unsupportedEnforcer{}
}

func (unsupportedEnforcer) IsRunning(string) bool {
	return false
}

func (unsupportedEnforcer) KillBrowser(processName string) error {
	return fmt.Errorf("enforcement isn't implemented on this platform yet")
}
