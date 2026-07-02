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

// Message is the envelope exchanged between background.js and the app, both
// over Native Messaging (ext <-> nmhost) and the local socket (nmhost <->
// app). Field names must stay in sync with the hand-written shapes in
// extension/background.js.
type Message struct {
	Type string `json:"type"`

	// hello (ext -> app)
	ExtID   string `json:"extId,omitempty"`
	Version string `json:"version,omitempty"`

	// setConfig (app -> ext)
	Rules []map[string]any `json:"rules,omitempty"`

	// setConfigAck, status (ext -> app)
	ConfigHash  string `json:"configHash,omitempty"`
	RulesActive int    `json:"rulesActive,omitempty"`

	// _bridgeHello (nmhost -> app only; synthetic, never sent by the
	// extension). Identifies which browser spawned this nmhost process.
	Browser string `json:"browser,omitempty"`
}

// Message.Type values.
const (
	TypeHello        = "hello"
	TypeSetConfig    = "setConfig"
	TypeSetConfigAck = "setConfigAck"
	TypeGetStatus    = "getStatus"
	TypeStatus       = "status"
	TypeBridgeHello  = "_bridgeHello"
)
