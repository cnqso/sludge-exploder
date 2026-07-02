package main

import (
	"os"
	"path/filepath"
	"runtime"
)

// BrowserDef describes one controllable browser we know how to detect and
// register a native-messaging host for. macOS only for now -- Windows/Linux
// parity is Stage 5, mirroring how /daemon stubs its non-darwin backends.
type BrowserDef struct {
	Key                string // stable identifier used across the UI and prefs
	Label              string // display name
	AppBundlePath      string // macOS: path used to detect installation
	ProcessName        string // must match nmhost's detectBrowser() output
	NativeMessagingDir string // where to write the host manifest
	Firefox            bool   // Firefox uses allowed_extensions, not allowed_origins
}

func knownBrowsers() []BrowserDef {
	if runtime.GOOS != "darwin" {
		return nil
	}
	home, _ := os.UserHomeDir()
	appSupport := filepath.Join(home, "Library", "Application Support")

	return []BrowserDef{
		{
			Key:                "chrome",
			Label:              "Google Chrome",
			AppBundlePath:      "/Applications/Google Chrome.app",
			ProcessName:        "Google Chrome",
			NativeMessagingDir: filepath.Join(appSupport, "Google", "Chrome", "NativeMessagingHosts"),
		},
		{
			Key:                "edge",
			Label:              "Microsoft Edge",
			AppBundlePath:      "/Applications/Microsoft Edge.app",
			ProcessName:        "Microsoft Edge",
			NativeMessagingDir: filepath.Join(appSupport, "Microsoft Edge", "NativeMessagingHosts"),
		},
		{
			Key:                "brave",
			Label:              "Brave Browser",
			AppBundlePath:      "/Applications/Brave Browser.app",
			ProcessName:        "Brave Browser",
			NativeMessagingDir: filepath.Join(appSupport, "BraveSoftware", "Brave-Browser", "NativeMessagingHosts"),
		},
		{
			Key:                "firefox",
			Label:              "Firefox",
			AppBundlePath:      "/Applications/Firefox.app",
			ProcessName:        "firefox",
			NativeMessagingDir: filepath.Join(appSupport, "Mozilla", "NativeMessagingHosts"),
			Firefox:            true,
		},
	}
}

func isInstalled(b BrowserDef) bool {
	_, err := os.Stat(b.AppBundlePath)
	return err == nil
}

func findBrowser(key string) (BrowserDef, bool) {
	for _, b := range knownBrowsers() {
		if b.Key == key {
			return b, true
		}
	}
	return BrowserDef{}, false
}
