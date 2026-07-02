package shared

import (
	"os"
	"path/filepath"
	"runtime"
)

// HostManifestPath is where the native-messaging-host manifest for this
// browser would live, whether or not it's actually been written yet.
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

// BrowserDef describes one controllable browser we know how to detect and
// register a native-messaging host for. Lives in shared (not just /app)
// because the daemon needs its own copy to make enforcement decisions
// without depending on the app being alive -- that's the whole point of the
// daemon outliving the UI. macOS only for now -- Windows/Linux parity is
// Stage 5, mirroring how /daemon's Enforcer stubs its non-darwin backends.
type BrowserDef struct {
	Key                string // stable identifier used across the UI and prefs
	Label              string // display name
	AppBundlePath      string // macOS: path used to detect installation
	ProcessName        string // must match nmhost's detectBrowser() output
	NativeMessagingDir string // where to write the host manifest
	Firefox            bool   // Firefox uses allowed_extensions, not allowed_origins
}

func KnownBrowsers() []BrowserDef {
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

func IsInstalled(b BrowserDef) bool {
	_, err := os.Stat(b.AppBundlePath)
	return err == nil
}

func FindBrowser(key string) (BrowserDef, bool) {
	for _, b := range KnownBrowsers() {
		if b.Key == key {
			return b, true
		}
	}
	return BrowserDef{}, false
}
