package shared

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

func KnownBrowsers() []BrowserDef {
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	localAppData := os.Getenv("LocalAppData")

	return []BrowserDef{
		{
			Key:         "chrome",
			Label:       "Google Chrome",
			ProcessName: "chrome.exe",
			InstallPaths: []string{
				// Chrome installs per-user by default on Windows, unlike
				// macOS -- check both locations.
				filepath.Join(localAppData, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(programFiles, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(programFilesX86, "Google", "Chrome", "Application", "chrome.exe"),
			},
			WindowsRegistryKey:  `Software\Google\Chrome\NativeMessagingHosts`,
			WindowsForcelistKey: `Software\Policies\Google\Chrome\ExtensionInstallForcelist`,
		},
		{
			Key:         "edge",
			Label:       "Microsoft Edge",
			ProcessName: "msedge.exe",
			InstallPaths: []string{
				filepath.Join(programFilesX86, "Microsoft", "Edge", "Application", "msedge.exe"),
				filepath.Join(programFiles, "Microsoft", "Edge", "Application", "msedge.exe"),
			},
			WindowsRegistryKey:  `Software\Microsoft\Edge\NativeMessagingHosts`,
			WindowsForcelistKey: `Software\Policies\Microsoft\Edge\ExtensionInstallForcelist`,
		},
		{
			Key:         "brave",
			Label:       "Brave Browser",
			ProcessName: "brave.exe",
			InstallPaths: []string{
				filepath.Join(programFiles, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
				filepath.Join(localAppData, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
			},
			WindowsRegistryKey: `Software\BraveSoftware\Brave-Browser\NativeMessagingHosts`,
			// Brave's policy vendor key -- unlike the NativeMessagingHosts
			// path above (confirmed against Brave's install branding),
			// I haven't been able to verify this against Brave's actual
			// enterprise policy docs from this environment. Treat as
			// best-effort; verify on the test device before relying on it.
			WindowsForcelistKey: `Software\Policies\BraveSoftware\Brave\ExtensionInstallForcelist`,
		},
		{
			Key:         "firefox",
			Label:       "Firefox",
			ProcessName: "firefox.exe",
			Firefox:     true,
			InstallPaths: []string{
				filepath.Join(programFiles, "Mozilla Firefox", "firefox.exe"),
			},
			WindowsRegistryKey: `Software\Mozilla\NativeMessagingHosts`,
		},
	}
}

func IsInstalled(b BrowserDef) bool {
	for _, p := range b.InstallPaths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// HostRegistered reports whether a browser has a native-messaging host
// registry value pointing at a manifest -- the Windows equivalent of
// macOS's HostManifestPath file check (see browsers_darwin.go). Same role:
// a real proxy for "the user set this browser up for Sludge Exploder," so
// the daemon's enforcement tick never targets a browser that was simply
// never configured for this tool.
func HostRegistered(b BrowserDef) bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, b.WindowsRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_, _, err = key.GetStringValue(NativeHostName)
	return err == nil
}
