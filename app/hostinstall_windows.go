package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"

	"github.com/cnqso/sludge-exploder/shared"
)

// registerBrowserHost writes the native-messaging-host manifest to a file
// under the app's own state directory, then points the browser at it via a
// per-user registry value -- Windows discovers native-messaging hosts
// through the registry, not a fixed directory of manifests like macOS (see
// hostinstall_darwin.go). HKEY_CURRENT_USER needs no admin elevation,
// matching the "not privileged, always removable by hand" posture carried
// over from Stage 3.
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

	manifestPath, err := windowsManifestPath(b)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(manifestPath), err)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return err
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, b.WindowsRegistryKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening registry key %s: %w", b.WindowsRegistryKey, err)
	}
	defer key.Close()
	return key.SetStringValue(shared.NativeHostName, manifestPath)
}

// windowsManifestPath is where this browser's manifest JSON file gets
// written -- unlike macOS, Windows doesn't need a fixed directory since the
// registry value points at wherever we choose to put it.
func windowsManifestPath(b shared.BrowserDef) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "SludgeExploder", "hosts", b.Key+".json"), nil
}
