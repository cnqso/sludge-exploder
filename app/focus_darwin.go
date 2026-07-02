package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"unsafe"
)

// focusSelf brings this app's window to the front, so the user sees it the
// moment a browser they care about enters the "at risk" state (see
// api.go's handleRiskFocus). Shells out to osascript rather than reaching
// for Cocoa/cgo directly -- same pattern the daemon already uses for
// pgrep/pkill (daemon/enforcer_darwin.go) and nmhost's browser detection.
// windowHandle is unused here -- macOS addresses by PID, not window handle
// (see focus_windows.go, which does need one).
func focusSelf(windowHandle unsafe.Pointer) {
	script := fmt.Sprintf(
		`tell application "System Events" to set frontmost of the first process whose unix id is %d to true`,
		os.Getpid(),
	)
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		log.Printf("sludge-exploder: bringing window to front: %v", err)
	}
}
