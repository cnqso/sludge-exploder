package main

import (
	"os"
	"path/filepath"
	"runtime"
)

// firefoxExtensionID must match extension/manifest.json's
// browser_specific_settings.gecko.id -- Firefox's ID is fixed there, so
// unlike Chrome-family browsers it never needs to be typed in by hand.
const firefoxExtensionID = "sludge-exploder@williamkelly"

// hostManifest is the native-messaging-host manifest JSON shape, identical
// on every platform. Where it's referenced from differs per OS -- a fixed
// directory of these files on macOS, a per-user registry value pointing at
// one on Windows -- see registerBrowserHost in hostinstall_darwin.go /
// hostinstall_windows.go.
type hostManifest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`
	AllowedOrigins    []string `json:"allowed_origins,omitempty"`
	AllowedExtensions []string `json:"allowed_extensions,omitempty"`
}

// nmhostPath returns the absolute path to the nmhost binary, expected next
// to the running app binary (both are built into bin/ by the Makefile).
// Native-messaging manifests require an absolute path, so exercising the
// config bridge needs `make build && ./bin/app` -- `go run ./app` builds to
// a transient temp path with no nmhost next to it.
func nmhostPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	name := "nmhost"
	if runtime.GOOS == "windows" {
		name = "nmhost.exe"
	}
	return filepath.Join(filepath.Dir(exe), name), nil
}
