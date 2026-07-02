package main

import (
	"log"
	"syscall"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)

// focusSelf brings this app's window to the front using the HWND
// webview_go's Window() already exposes (captured once at startup, see
// main.go). Calling user32.dll directly via syscall.NewLazyDLL is pure Go
// -- no cgo, same "avoid cgo" bar as the osascript approach on macOS
// (focus_darwin.go). UNTESTED: no Windows runtime access when this was
// written -- see docs/OPEN_TEST_QUESTIONS.md.
func focusSelf(windowHandle unsafe.Pointer) {
	if windowHandle == nil {
		log.Printf("sludge-exploder: bringing window to front: no window handle captured")
		return
	}
	procSetForegroundWindow.Call(uintptr(windowHandle))
}
