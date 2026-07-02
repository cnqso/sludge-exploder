package main

// Enforcer is the per-OS mechanism for checking and closing a browser
// process. It's deliberately narrow -- see docs/ENFORCEMENT.md §6: Stage 3
// is "friction, not lockdown," so the only mechanism here is closing a
// process, nothing resembling self-defense or re-spawn-on-kill.
type Enforcer interface {
	IsRunning(processName string) bool
	KillBrowser(processName string) error
}
