package main

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/cnqso/sludge-exploder/shared"
)

// EnforcementState is the runtime on/off switch: --enforce at startup,
// toggleable at any time via the control channel's SetEnforcement command.
// This is the safety override from the Stage 3 plan -- it works
// independently of lock state (flipping it off doesn't unlock anything or
// stop the extension's CSS blocking, it only disables process-killing) so
// it's always available as a "stop everything" button if a test misbehaves.
// Not persisted: a daemon restart resets to the conservative --enforce
// default rather than whatever was last toggled at runtime.
type EnforcementState struct {
	mu      sync.Mutex
	enabled bool
}

func NewEnforcementState(initial bool) *EnforcementState {
	return &EnforcementState{enabled: initial}
}

func (e *EnforcementState) Enabled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.enabled
}

func (e *EnforcementState) Set(enabled bool) {
	e.mu.Lock()
	e.enabled = enabled
	e.mu.Unlock()
}

// sessionTracker gives a browser a startup grace period measured from when
// it launches, not from when it happens to first look closeable -- this is
// how Cold Turkey does it, and it matters for real recovery: closing an
// extension-disabled browser the instant a lock starts doesn't give you
// time to re-enable it or flip the app's Enforcement toggle off. A browser
// gets `grace` from the moment it's first observed running (in the same
// continuous run -- closing and reopening it starts a fresh session) to get
// its extension connected; after that window elapses, it's closed on the
// very first tick it's found missing, no further waiting.
//
// This is deliberately NOT per-site allowlisting -- the kill mechanism
// operates on the whole browser process, so it has no way to "let the Web
// Store through but nothing else." Per-site rules live in Tier 1 (the
// extension's CSS blocking, config.js's existing domain/path/allowWindow
// schema); Tier 3's kill switch is all-or-nothing for a given process. The
// startup grace period plus the Enforcement toggle are the actual recovery
// tools at this layer.
type sessionTracker struct {
	mu         sync.Mutex
	launchedAt map[string]time.Time
	warned     map[string]bool
}

func newSessionTracker() *sessionTracker {
	return &sessionTracker{launchedAt: make(map[string]time.Time), warned: make(map[string]bool)}
}

// observe records whether `key` is currently running. The first time it's
// seen running starts its session timer; when it's not running, its session
// ends (so a future relaunch starts a fresh grace period). Called every
// tick regardless of lock state -- a browser's grace period is measured
// from when it launches, not from when a lock happens to start.
func (t *sessionTracker) observe(key string, running bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !running {
		delete(t.launchedAt, key)
		delete(t.warned, key)
		return
	}
	if _, ok := t.launchedAt[key]; !ok {
		t.launchedAt[key] = time.Now()
	}
}

// warnOnce reports true the first time it's called for `key` since its
// session began, false on every call after that -- used to log a single
// explanatory line instead of either silence or 5s-interval spam.
func (t *sessionTracker) warnOnce(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.warned[key] {
		return false
	}
	t.warned[key] = true
	return true
}

// pastGracePeriod reports whether `key`'s current session has run for at
// least `grace` -- i.e. whether its startup grace period has elapsed and it
// should be closed on the very next tick it's found missing/uncontrolled.
func (t *sessionTracker) pastGracePeriod(key string, grace time.Duration) bool {
	return t.graceRemaining(key, grace) <= 0
}

// graceRemaining is how much of `key`'s startup grace period is left. Used
// to report "at risk" state over the control channel so the app can warn
// the user (and bring itself to the front) during the window, not just
// silently either wait or close.
func (t *sessionTracker) graceRemaining(key string, grace time.Duration) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	launched, ok := t.launchedAt[key]
	if !ok {
		return grace
	}
	remaining := grace - time.Since(launched)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// riskInfo is one browser's "at risk" state: grace period running, not yet
// enforced. Populated by the enforcement tick (which already computes all
// the inputs) and read by the control channel's GetStatus, so the app can
// warn the user -- and bring itself to the front, see app/focus.go -- while
// there's still time to react, not just log the eventual close.
type riskInfo struct {
	AtRisk         bool
	GraceRemaining time.Duration
}

// riskCache is the thread-safe handoff between the enforcement tick
// (writer, every 5s) and control-channel status responses (reader, on
// every app poll) -- reading never triggers extra IsRunning/pgrep calls of
// its own.
type riskCache struct {
	mu   sync.Mutex
	data map[string]riskInfo
}

func newRiskCache() *riskCache {
	return &riskCache{data: make(map[string]riskInfo)}
}

func (c *riskCache) set(key string, info riskInfo) {
	c.mu.Lock()
	c.data[key] = info
	c.mu.Unlock()
}

func (c *riskCache) clear(key string) {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
}

func (c *riskCache) get(key string) riskInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.data[key] // zero value (AtRisk: false) if absent
}

// EnforcementLoop runs the periodic tick: while LOCKED, close a
// controllable browser whose extension heartbeat has gone missing (past its
// startup grace period), and close any explicitly-designated "uncontrolled"
// browser that's running (likewise past its grace period). interval matches
// the heartbeat cadence (5s) -- no tighter, no hammering.
func EnforcementLoop(lock *LockManager, hb *HeartbeatServer, enforcer Enforcer, state *EnforcementState, uncontrolled []string, interval, gracePeriod time.Duration, risk *riskCache) {
	sessions := newSessionTracker()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		tick(lock, hb, enforcer, state, uncontrolled, sessions, gracePeriod, risk)
	}
}

func tick(lock *LockManager, hb *HeartbeatServer, enforcer Enforcer, state *EnforcementState, uncontrolled []string, sessions *sessionTracker, gracePeriod time.Duration, risk *riskCache) {
	lockState, _ := lock.State()
	locked := lockState == shared.LockStateLocked

	for _, b := range shared.KnownBrowsers() {
		running := enforcer.IsRunning(b.ProcessName)
		sessions.observe(b.ProcessName, running) // always, even unlocked

		if !locked || !running || hb.IsAlive(b.ProcessName) {
			risk.clear(b.ProcessName)
			continue
		}
		// Both checks matter and neither is sufficient alone:
		//   - HostRegistered alone isn't enough because AutoRegisterHosts
		//     writes a manifest for every *installed* browser, not just
		//     ones the user actually loaded the extension into -- a daily
		//     driver browser can have a manifest and still have never run
		//     the extension.
		//   - EverAlive alone isn't enough because it only reflects this
		//     daemon process's own lifetime, not user intent.
		// Together: only a browser that was both set up for this tool AND
		// has actually heartbeated at least once (then gone quiet) is
		// "missing." A browser that's simply never been used with the
		// extension is left alone, full stop.
		if !shared.HostRegistered(b) || !hb.EverAlive(b.ProcessName) {
			risk.clear(b.ProcessName)
			if sessions.warnOnce(b.ProcessName) {
				log.Printf("sludge-exploder: %s is running during a lock but its extension has never connected this session -- not enforcing (enable it in the browser first)", b.Label)
			}
			continue
		}

		remaining := sessions.graceRemaining(b.ProcessName, gracePeriod)
		risk.set(b.ProcessName, riskInfo{AtRisk: true, GraceRemaining: remaining})
		if remaining > 0 {
			continue // still within this session's startup grace window
		}
		enforce(enforcer, state, b.ProcessName, "extension heartbeat missing")
	}

	for _, raw := range uncontrolled {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		running := enforcer.IsRunning(name)
		sessions.observe(name, running)

		if !locked || !running {
			continue
		}
		if !sessions.pastGracePeriod(name, gracePeriod) {
			continue
		}
		enforce(enforcer, state, name, "uncontrolled browser running during a lock")
	}
}

// enforce is the single choke point every kill decision passes through, so
// the enabled/disabled gate can never be bypassed by a code path that
// forgets to check it.
func enforce(enforcer Enforcer, state *EnforcementState, processName, reason string) {
	if !state.Enabled() {
		log.Printf("sludge-exploder: [log-only] would close %s (%s)", processName, reason)
		return
	}
	log.Printf("sludge-exploder: closing %s (%s)", processName, reason)
	if err := enforcer.KillBrowser(processName); err != nil {
		log.Printf("sludge-exploder: closing %s: %v", processName, err)
	}
}
