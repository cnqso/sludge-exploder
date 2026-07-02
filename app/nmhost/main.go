// Command nmhost is the Native Messaging host Chrome/Firefox spawn per
// connectNative() call from extension/background.js. It has no business
// logic of its own: it's a dumb relay between Chrome's length-prefixed
// stdio framing and the daemon's local IPC channel
// (shared.HeartbeatSocketPath() -- a Unix socket on macOS, a named pipe on
// Windows; see shared/ipc.go).
//
// It's a separate binary, not a flag on the app, because native-messaging
// host manifests invoke "path" directly with fixed argv the browser
// controls -- there's no way to pass a custom flag through that contract.
//
// Stage 2 pointed this at the app's socket; Stage 3 moves the target to the
// daemon so the heartbeat reaches the process that actually enforces
// anything (docs/ENFORCEMENT.md §5.1) -- the relay/framing logic below is
// unchanged, only the dial target moved.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cnqso/sludge-exploder/shared"
)

func main() {
	// stdout is reserved for native-messaging framing to the browser; never
	// write logs there.
	log.SetOutput(os.Stderr)

	path, err := shared.HeartbeatSocketPath()
	if err != nil {
		log.Printf("nmhost: resolving socket path: %v", err)
		return
	}

	conn, err := shared.DialIPC(path, 2*time.Second)
	if err != nil {
		// The daemon isn't running (yet). Exit quietly -- background.js's
		// own reconnect loop will spawn us again shortly.
		return
	}
	defer conn.Close()

	hello := shared.Message{Type: shared.TypeBridgeHello, Browser: detectBrowser()}
	if data, err := json.Marshal(hello); err == nil {
		conn.Write(append(data, '\n'))
	}

	done := make(chan struct{})
	var once sync.Once
	closeAll := func() {
		once.Do(func() {
			conn.Close()
			close(done)
		})
	}

	go func() {
		relaySocketToStdout(conn)
		closeAll()
	}()
	go func() {
		relayStdinToSocket(conn)
		closeAll()
	}()

	<-done
}

func relaySocketToStdout(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if err := shared.WriteNativeMessage(os.Stdout, scanner.Bytes()); err != nil {
			return
		}
	}
}

func relayStdinToSocket(conn net.Conn) {
	for {
		payload, err := shared.ReadNativeMessage(os.Stdin)
		if err != nil {
			return
		}
		if _, err := conn.Write(append(payload, '\n')); err != nil {
			return
		}
	}
}

// detectBrowser identifies which browser spawned this process by looking up
// its parent process's command name. Best-effort; the returned name must
// match a BrowserDef.ProcessName in shared.KnownBrowsers() for that OS.
func detectBrowser() string {
	switch runtime.GOOS {
	case "darwin":
		return detectBrowserDarwin()
	case "windows":
		return detectBrowserWindows()
	default:
		return ""
	}
}

func detectBrowserDarwin() string {
	out, err := exec.Command("ps", "-o", "comm=", "-p", fmt.Sprint(os.Getppid())).Output()
	if err != nil {
		return ""
	}
	comm := strings.TrimSpace(string(out))
	if comm == "" {
		return ""
	}
	return filepath.Base(comm)
}

func detectBrowserWindows() string {
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", os.Getppid()), "/NH", "/FO", "CSV").Output()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(out))
	fields := strings.Split(line, ",")
	if len(fields) == 0 {
		return ""
	}
	// tasklist's CSV output quotes each field, e.g. "chrome.exe","1234",...
	return strings.Trim(fields[0], `"`)
}
