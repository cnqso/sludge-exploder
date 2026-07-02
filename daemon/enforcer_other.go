//go:build !darwin

package main

import "fmt"

// Windows/Linux parity is Stage 5 (see enforcer_darwin.go and
// docs/ENFORCEMENT.md §4.3's "one daemon, two backends" plan).
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
