// Package shared holds types and constants used by /app and /app/nmhost (and
// later /daemon): the IPC message envelope exchanged between the extension's
// background script and the app, relayed over Native Messaging by nmhost and
// over a local Unix socket by the app. See docs/ENFORCEMENT.md §5.
package shared

// Version is the current protocol/schema version, bumped when the shapes
// below change in an incompatible way.
const Version = "0.0.0"

// NativeHostName must match extension/background.js's connectNative(...)
// argument and the "name" field written into the native-messaging-host
// manifest by app/hostinstall.go.
const NativeHostName = "com.sludgeexploder.host"

// ChromeExtensionID is pinned via the "key" field in extension/manifest.json
// (an embedded RSA public key), so it's fixed and known ahead of time
// instead of varying by install path -- this is what lets the app
// auto-register every Chrome-family browser's native-messaging host with no
// manual "paste the extension ID" step. Firefox's ID is separately pinned
// via browser_specific_settings.gecko.id and needs no such constant.
//
// The corresponding private key was used only to compute this ID and was
// deliberately not persisted anywhere in the repo (a plain "key" field
// doesn't require it for anything at runtime). If this identity ever needs
// to sign a .crx for real distribution, Stage 5's proper key management
// generates a new one -- that's a separate, already-open question in
// docs/ENFORCEMENT.md §9.
const ChromeExtensionID = "iaebhkbbkbjonipfifaidlcmdkgaijik"

// Message is the envelope exchanged both on the heartbeat channel
// (extension <-> nmhost <-> daemon) and the control channel (app <->
// daemon). Not every field applies to every Type; see the comments below.
// Field names on the heartbeat side must stay in sync with the hand-written
// shapes in extension/background.js.
type Message struct {
	Type string `json:"type"`

	// hello, heartbeat (ext -> daemon)
	ExtID   string `json:"extId,omitempty"`
	Version string `json:"version,omitempty"`

	// setConfig (app -> daemon -> ext)
	Rules []map[string]any `json:"rules,omitempty"`

	// setConfigAck, status, heartbeat (ext -> daemon)
	ConfigHash  string `json:"configHash,omitempty"`
	RulesActive int    `json:"rulesActive,omitempty"`

	// _bridgeHello (nmhost -> daemon only; synthetic, never sent by the
	// extension). Identifies which browser spawned this nmhost process.
	Browser string `json:"browser,omitempty"`

	// Heartbeat replies (daemon -> ext) and control status responses
	// (daemon -> app).
	LockState     string                   `json:"lockState,omitempty"`
	Until         string                   `json:"until,omitempty"` // RFC3339; empty if UNLOCKED
	ConfigToApply []map[string]any         `json:"configToApply,omitempty"`
	Enforcement   bool                     `json:"enforcement,omitempty"`
	Browsers      []BrowserHeartbeatStatus `json:"browsers,omitempty"`
	Risk          []BrowserRisk            `json:"risk,omitempty"`

	// Control channel only (app -> daemon).
	Token           string `json:"token,omitempty"`
	DurationSeconds int    `json:"durationSeconds,omitempty"`
	Enabled         bool   `json:"enabled,omitempty"`

	// error (daemon -> app; e.g. StopLock refusal, bad token)
	Error string `json:"error,omitempty"`
}

// BrowserHeartbeatStatus is one browser's live state as tracked by the
// daemon's heartbeat server, returned in control-channel status responses.
type BrowserHeartbeatStatus struct {
	Browser     string `json:"browser"`
	Connected   bool   `json:"connected"`
	Alive       bool   `json:"alive"` // connected AND heartbeating within threshold
	ExtID       string `json:"extId,omitempty"`
	Version     string `json:"version,omitempty"`
	ConfigHash  string `json:"configHash,omitempty"`
	RulesActive int    `json:"rulesActive,omitempty"`
}

// BrowserRisk is one browser's "at risk" state: a lock is active, this
// browser's extension has gone missing, and it's still within its startup
// grace period (see daemon/enforce.go's sessionTracker) -- not yet closed,
// but will be once the countdown reaches zero unless it reconnects.
type BrowserRisk struct {
	Browser               string `json:"browser"` // process name, e.g. "Google Chrome"
	Label                 string `json:"label"`   // display name
	GraceRemainingSeconds int    `json:"graceRemainingSeconds"`
}

// Message.Type values.
const (
	// Heartbeat channel (extension <-> daemon, via nmhost).
	TypeHello        = "hello"
	TypeHeartbeat    = "heartbeat"
	TypeSetConfig    = "setConfig"
	TypeSetConfigAck = "setConfigAck"
	TypeGetStatus    = "getStatus"
	TypeStatus       = "status"
	TypeBridgeHello  = "_bridgeHello"

	// Control channel (app <-> daemon).
	TypeStartLock      = "startLock"
	TypeStopLock       = "stopLock"
	TypeSetEnforcement = "setEnforcement"
	TypeLockStatus     = "lockStatus"
	TypeError          = "error"
)

// Lock states, shared verbatim between daemon/lock.go and the JS/UI side.
const (
	LockStateUnlocked = "UNLOCKED"
	LockStateLocked   = "LOCKED"
)
