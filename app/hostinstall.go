package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cnqso/sludge-exploder/shared"
)

// firefoxExtensionID must match extension/manifest.json's
// browser_specific_settings.gecko.id -- Firefox's ID is fixed there, so
// unlike Chrome-family browsers it never needs to be typed in by hand.
const firefoxExtensionID = "sludge-exploder@williamkelly"

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
	return filepath.Join(filepath.Dir(exe), "nmhost"), nil
}

// registerBrowserHost writes (or overwrites) the native-messaging-host
// manifest for one browser. Both extension IDs are fixed ahead of time
// (shared.ChromeExtensionID via manifest.json's "key" field, Firefox's via
// browser_specific_settings.gecko.id), so this needs no input from the
// user -- it's safe to call automatically for every detected browser.
func registerBrowserHost(b shared.BrowserDef) error {
	nmhost, err := nmhostPath()
	if err != nil {
		return fmt.Errorf("locating nmhost binary: %w", err)
	}
	if _, err := os.Stat(nmhost); err != nil {
		return fmt.Errorf("nmhost binary not found at %s -- run `make build` first", nmhost)
	}

	manifest := hostManifest{
		Name:        shared.NativeHostName,
		Description: "Sludge Exploder config bridge",
		Path:        nmhost,
		Type:        "stdio",
	}

	if b.Firefox {
		manifest.AllowedExtensions = []string{firefoxExtensionID}
	} else {
		manifest.AllowedOrigins = []string{fmt.Sprintf("chrome-extension://%s/", shared.ChromeExtensionID)}
	}

	if err := os.MkdirAll(b.NativeMessagingDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", b.NativeMessagingDir, err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(shared.HostManifestPath(b), data, 0o644)
}
