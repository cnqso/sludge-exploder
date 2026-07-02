package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cnqso/sludge-exploder/shared"
)

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
