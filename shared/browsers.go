package shared

// BrowserDef describes one controllable browser we know how to detect and
// register a native-messaging host for. Lives in shared (not just /app)
// because the daemon needs its own copy to make enforcement decisions
// without depending on the app being alive -- that's the whole point of the
// daemon outliving the UI.
//
// KnownBrowsers/IsInstalled/HostRegistered are implemented per-OS
// (browsers_darwin.go, browsers_windows.go) since detection and
// native-messaging registration work completely differently on each
// platform (a fixed app bundle path + directory of JSON manifests on
// macOS; multiple possible install paths + a per-user registry value on
// Windows) -- not every field below applies on every platform.
type BrowserDef struct {
	Key         string // stable identifier used across the UI and prefs
	Label       string // display name
	ProcessName string // must match nmhost's detectBrowser() output
	Firefox     bool   // Firefox uses allowed_extensions, not allowed_origins

	// macOS only.
	AppBundlePath      string // path used to detect installation
	NativeMessagingDir string // where to write the host manifest

	// Windows only.
	InstallPaths        []string // candidate install paths; installed if any exist
	WindowsRegistryKey  string   // e.g. `Software\Google\Chrome\NativeMessagingHosts`
	WindowsForcelistKey string   // e.g. `Software\Policies\Google\Chrome\ExtensionInstallForcelist`; Chrome-family only, empty for Firefox (no such mechanism)
}

func FindBrowser(key string) (BrowserDef, bool) {
	for _, b := range KnownBrowsers() {
		if b.Key == key {
			return b, true
		}
	}
	return BrowserDef{}, false
}
