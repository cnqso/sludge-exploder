package shared

import (
	"os"
	"path/filepath"
)

func KnownBrowsers() []BrowserDef {
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

func IsInstalled(b BrowserDef) bool {
	_, err := os.Stat(b.AppBundlePath)
	return err == nil
}

// HostManifestPath is where the native-messaging-host manifest for this
// browser would live, whether or not it's actually been written yet.
// macOS only -- Windows registers a manifest file at an arbitrary path via
// a registry value instead of a fixed directory, see hostRegisteredWindows
// in browsers_windows.go.
func HostManifestPath(b BrowserDef) string {
	return filepath.Join(b.NativeMessagingDir, NativeHostName+".json")
}

// HostRegistered reports whether a browser has an installed host manifest
// -- a real proxy for "the user set this browser up for Sludge Exploder,"
// not just "this browser happens to be installed on the machine." The
// daemon's enforcement tick uses this to decide which browsers it's
// actually responsible for policing: an installed-but-never-configured
// browser (e.g. a daily-driver browser the user never loaded the extension
// into) must never be targeted just because it has no heartbeat -- it was
// never expected to have one.
func HostRegistered(b BrowserDef) bool {
	_, err := os.Stat(HostManifestPath(b))
	return err == nil
}
