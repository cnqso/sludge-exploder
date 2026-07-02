package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cnqso/sludge-exploder/shared"
)

// App holds the state backing the Go functions bound into the webview (see
// main.go). All access goes through the mutex since webview_go's Bind
// callbacks aren't documented as single-threaded. Stage 3: App no longer
// holds any bridge/heartbeat state itself -- that lives in the daemon now.
// App is a thin client of it (daemon field, lazily resolved via
// daemonClient() below so app/daemon launch order doesn't matter -- same
// self-healing spirit as the extension's own reconnect loop).
type App struct {
	mu     sync.Mutex
	prefs  Prefs
	daemon *DaemonClient

	// focusedForRisk tracks which browsers we've already self-focused for,
	// so handleRiskFocus brings the window forward once per new "at risk"
	// occurrence, not on every 1s status poll while it's still ongoing.
	focusedForRisk map[string]bool
}

// daemonClient returns a connected client, attempting to (re)create one if
// none exists yet -- e.g. the app started before the daemon did, so the
// control token file didn't exist at startup. Returns nil if the daemon
// still isn't reachable.
func (a *App) daemonClient() *DaemonClient {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.daemon != nil {
		return a.daemon
	}
	client, err := NewDaemonClient()
	if err != nil {
		return nil
	}
	a.daemon = client
	return a.daemon
}

// BrowserInfo is what the setup helper and status panel poll: filesystem
// detection, whether we've written a host manifest, and live heartbeat
// state fetched from the daemon.
type BrowserInfo struct {
	Key            string `json:"key"`
	Label          string `json:"label"`
	Installed      bool   `json:"installed"`
	HostRegistered bool   `json:"hostRegistered"`
	Connected      bool   `json:"connected"`
	Alive          bool   `json:"alive"`
	ExtID          string `json:"extId,omitempty"`
	Version        string `json:"version,omitempty"`
	ConfigHash     string `json:"configHash,omitempty"`
	RulesActive    int    `json:"rulesActive,omitempty"`
}

// LockStatus is what the UI polls for the countdown, the "attempt early
// unlock" affordance, the enforcement toggle, and any at-risk warnings.
type LockStatus struct {
	State       string               `json:"state"` // "UNLOCKED" or "LOCKED"
	Until       string               `json:"until,omitempty"`
	Enforcement bool                 `json:"enforcement"`
	DaemonUp    bool                 `json:"daemonUp"`
	AtRisk      []shared.BrowserRisk `json:"atRisk,omitempty"`
}

// AutoRegisterHosts writes a native-messaging host manifest for every
// installed, not-yet-registered browser. Both Chrome-family and Firefox
// extension IDs are fixed ahead of time (see shared.ChromeExtensionID), so
// this needs no user input and is safe to run unconditionally on startup --
// onboarding should require loading the unpacked extension and nothing
// else.
func (a *App) AutoRegisterHosts() {
	for _, b := range shared.KnownBrowsers() {
		if !shared.IsInstalled(b) || shared.HostRegistered(b) {
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
// list, and sends it to the daemon, which pushes it to every connected
// browser's extension.
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

	if client := a.daemonClient(); client != nil {
		if _, err := client.SetConfig(rules); err != nil {
			log.Printf("sludge-exploder: pushing config to daemon: %v", err)
		}
	}
	return p, nil
}

func (a *App) GetDetectedBrowsers() ([]BrowserInfo, error) {
	return a.mergedBrowserInfo(), nil
}

func (a *App) GetConnectionStatus() ([]BrowserInfo, error) {
	return a.mergedBrowserInfo(), nil
}

func (a *App) mergedBrowserInfo() []BrowserInfo {
	byBrowser := map[string]shared.BrowserHeartbeatStatus{}
	if client := a.daemonClient(); client != nil {
		if status, err := client.GetStatus(); err == nil {
			for _, s := range status.Browsers {
				if s.Browser != "" {
					byBrowser[s.Browser] = s
				}
			}
		}
	}

	var out []BrowserInfo
	for _, b := range shared.KnownBrowsers() {
		if !shared.IsInstalled(b) {
			continue
		}
		info := BrowserInfo{
			Key:            b.Key,
			Label:          b.Label,
			Installed:      true,
			HostRegistered: shared.HostRegistered(b),
		}
		if st, ok := byBrowser[b.ProcessName]; ok {
			info.Connected = st.Connected
			info.Alive = st.Alive
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
	b, ok := shared.FindBrowser(browserKey)
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

// GetLockStatus is polled by the UI for the countdown and enforcement
// toggle. DaemonUp is false (rather than an error) when the daemon isn't
// reachable, so the UI can show "daemon not running" instead of a raw
// connection error.
func (a *App) GetLockStatus() (LockStatus, error) {
	client := a.daemonClient()
	if client == nil {
		return LockStatus{State: shared.LockStateUnlocked}, nil
	}
	status, err := client.GetStatus()
	if err != nil {
		return LockStatus{State: shared.LockStateUnlocked}, nil
	}

	a.handleRiskFocus(status.Risk)

	return LockStatus{
		State:       status.LockState,
		Until:       status.Until,
		Enforcement: status.Enforcement,
		DaemonUp:    true,
		AtRisk:      status.Risk,
	}, nil
}

// handleRiskFocus brings the app window to the front the moment a browser
// newly enters the "at risk" state (grace period counting down), not on
// every 1s poll for as long as it stays that way -- one focus-steal per
// incident, not a stream of them.
func (a *App) handleRiskFocus(risk []shared.BrowserRisk) {
	a.mu.Lock()
	newlyAtRisk := false
	current := make(map[string]bool, len(risk))
	for _, r := range risk {
		current[r.Browser] = true
		if !a.focusedForRisk[r.Browser] {
			newlyAtRisk = true
		}
	}
	a.focusedForRisk = current
	a.mu.Unlock()

	if newlyAtRisk {
		focusSelf()
	}
}

// ConfirmLock starts a real, daemon-enforced lock (Stage 3 -- Stage 2's
// version only recorded intent). Also persists the confirmed duration/
// summary locally so the UI has something to show even if the daemon is
// briefly unreachable.
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

	client := a.daemonClient()
	if client == nil {
		return intent, fmt.Errorf("daemon is not running -- start it with `./bin/daemon` first")
	}
	if _, err := client.StartLock(time.Duration(durationMinutes) * time.Minute); err != nil {
		return intent, err
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

// AttemptUnlock always tries StopLock and returns whatever the daemon says
// -- while LOCKED, that's an explicit refusal, shown as-is in the UI so the
// mechanism is visibly real rather than just a disabled button.
func (a *App) AttemptUnlock() (LockStatus, error) {
	client := a.daemonClient()
	if client == nil {
		return LockStatus{State: shared.LockStateUnlocked}, fmt.Errorf("daemon is not running")
	}
	resp, err := client.StopLock()
	if err != nil {
		return LockStatus{}, err
	}
	return LockStatus{State: resp.LockState, Until: resp.Until, Enforcement: resp.Enforcement, DaemonUp: true}, nil
}

// SetEnforcement is the safety override described in the Stage 3 plan:
// works regardless of lock state, always available in the UI.
func (a *App) SetEnforcement(enabled bool) (LockStatus, error) {
	client := a.daemonClient()
	if client == nil {
		return LockStatus{State: shared.LockStateUnlocked}, fmt.Errorf("daemon is not running")
	}
	resp, err := client.SetEnforcement(enabled)
	if err != nil {
		return LockStatus{}, err
	}
	return LockStatus{State: resp.LockState, Until: resp.Until, Enforcement: resp.Enforcement, DaemonUp: true}, nil
}
