// Package shared: this file declares the cross-platform IPC seam every
// socket/pipe caller (daemon/heartbeat.go, daemon/control.go,
// app/daemonclient.go, app/nmhost/main.go) uses instead of calling
// net.Listen/net.DialTimeout with "unix" directly. macOS has Unix domain
// sockets; Windows has none (Go's net package doesn't support Windows
// named pipes at all), so this seam is what keeps every caller's code
// identical on both platforms -- see ipc_darwin.go and ipc_windows.go for
// the two implementations.
package shared

import (
	"net"
	"time"
)

// ListenIPC starts listening on the local IPC channel at path (a
// filesystem socket path on macOS, a named-pipe name on Windows -- see
// HeartbeatSocketPath/ControlSocketPath in paths_darwin.go/paths_windows.go).
func ListenIPC(path string) (net.Listener, error) {
	return listenIPC(path)
}

// DialIPC connects to the local IPC channel at path, timing out after
// timeout if it can't connect (e.g. nothing is listening yet).
func DialIPC(path string, timeout time.Duration) (net.Conn, error) {
	return dialIPC(path, timeout)
}
