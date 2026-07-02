//go:build !darwin && !windows

package main

import "unsafe"

// No-op on platforms without an implementation yet (see focus_darwin.go,
// focus_windows.go).
func focusSelf(windowHandle unsafe.Pointer) {}
