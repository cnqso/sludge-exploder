package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// App holds the state backing the Go functions bound into the webview (see
// main.go). All access goes through the mutex since webview_go's Bind
// callbacks aren't documented as single-threaded.
type App struct {
	mu     sync.Mutex
	prefs  Prefs
	socket *SocketServer
}

// BrowserInfo is what the setup helper and status panel poll: filesystem
// detection, whether we've written a host manifest, and live connection
// state merged from the socket server.
type BrowserInfo struct {
	Key            string `json:"key"`
	Label          string `json:"label"`
	Installed      bool   `json:"installed"`
	HostRegistered bool   `json:"hostRegistered"`
	Connected      bool   `json:"connected"`
	ExtID          string `json:"extId,omitempty"`
	Version        string `json:"version,omitempty"`
	ConfigHash     string `json:"configHash,omitempty"`
	RulesActive    int    `json:"rulesActive,omitempty"`
}

// AutoRegisterHosts writes a native-messaging host manifest for every
// installed, not-yet-registered browser. Both Chrome-family and Firefox
// extension IDs are fixed ahead of time (see shared.ChromeExtensionID), so
// this needs no user input and is safe to run unconditionally on startup --
// onboarding should require loading the unpacked extension and nothing
// else.
func (a *App) AutoRegisterHosts() {
	for _, b := range knownBrowsers() {
		if !isInstalled(b) || hostRegistered(b) {
			continue
		}
		if err := registerBrowserHost(b); err != nil {
			log.Printf("sludge-exploder: auto-registering %s: %v", b.Label, err)
		}
	}
}

func (a *App) GetCatalog() ([]CatalogApp, error) {
	return starterCatalog(), nil
}

func (a *App) GetPrefs() (Prefs, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.prefs, nil
}

// SavePrefs persists the UI's selections, rebuilds the SLUDGE_CONFIG rule
// list, and pushes it to every connected browser's extension.
func (a *App) SavePrefs(p Prefs) (Prefs, error) {
	if p.Selections == nil {
		p.Selections = map[string]map[string]bool{}
	}

	a.mu.Lock()
	a.prefs = p
	rules := buildConfig(a.prefs, starterCatalog())
	a.mu.Unlock()

	if err := savePrefsToDisk(p); err != nil {
		return p, err
	}
	a.socket.PushSetConfig(rules)
	return p, nil
}

func (a *App) GetDetectedBrowsers() ([]BrowserInfo, error) {
	return a.mergedBrowserInfo(), nil
}

func (a *App) GetConnectionStatus() ([]BrowserInfo, error) {
	return a.mergedBrowserInfo(), nil
}

func (a *App) mergedBrowserInfo() []BrowserInfo {
	byBrowser := map[string]BrowserStatus{}
	for _, s := range a.socket.Snapshot() {
		if s.Browser != "" {
			byBrowser[s.Browser] = s
		}
	}

	var out []BrowserInfo
	for _, b := range knownBrowsers() {
		if !isInstalled(b) {
			continue
		}
		info := BrowserInfo{
			Key:            b.Key,
			Label:          b.Label,
			Installed:      true,
			HostRegistered: hostRegistered(b),
		}
		if st, ok := byBrowser[b.ProcessName]; ok {
			info.Connected = true
			info.ExtID = st.ExtID
			info.Version = st.Version
			info.ConfigHash = st.ConfigHash
			info.RulesActive = st.RulesActive
		}
		out = append(out, info)
	}
	return out
}

// RegisterBrowserHost is a manual retry, exposed to the UI in case
// auto-registration failed (e.g. a permissions error writing to the
// browser's NativeMessagingHosts directory).
func (a *App) RegisterBrowserHost(browserKey string) (BrowserInfo, error) {
	b, ok := findBrowser(browserKey)
	if !ok {
		return BrowserInfo{}, fmt.Errorf("unknown browser %q", browserKey)
	}
	if err := registerBrowserHost(b); err != nil {
		return BrowserInfo{}, err
	}
	for _, info := range a.mergedBrowserInfo() {
		if info.Key == browserKey {
			return info, nil
		}
	}
	return BrowserInfo{}, nil
}

// ConfirmLock is Stage 2's fake Lock: it records intent and shows a summary
// but enforces nothing (the daemon that makes this real lands in Stage 3).
func (a *App) ConfirmLock(durationMinutes int) (LockIntent, error) {
	a.mu.Lock()
	blockedApps := 0
	for _, parts := range a.prefs.Selections {
		for _, enabled := range parts {
			if enabled {
				blockedApps++
				break
			}
		}
	}
	a.mu.Unlock()

	intent := LockIntent{
		DurationMinutes: durationMinutes,
		Summary:         fmt.Sprintf("Blocking %d app(s) for %d minute(s)", blockedApps, durationMinutes),
		ConfirmedAt:     time.Now().Format(time.RFC3339),
	}

	a.mu.Lock()
	a.prefs.DurationMinutes = durationMinutes
	a.prefs.LastLockIntent = &intent
	p := a.prefs
	a.mu.Unlock()

	if err := savePrefsToDisk(p); err != nil {
		return intent, err
	}
	return intent, nil
}
