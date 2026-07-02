package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/cnqso/sludge-exploder/shared"
)

// heartbeatMissingAfter is roughly 3 missed 5s beats. A bridge whose
// connection closes entirely (extension disabled/removed -- Chrome tears
// down the port immediately) is detected the instant its read loop ends,
// no waiting required; this threshold only covers the rarer case of a
// connection staying open without heartbeating.
const heartbeatMissingAfter = 15 * time.Second

// bridgeConn tracks one connected nmhost relay: one browser's extension.
type bridgeConn struct {
	conn          net.Conn
	writeMu       sync.Mutex // serializes writes; net.Conn isn't safe for concurrent Write
	browser       string
	extID         string
	version       string
	configHash    string
	rulesActive   int
	lastHeartbeat time.Time
}

// HeartbeatServer is the daemon side of the extension <-> daemon channel
// (extension/background.js --connectNative--> app/nmhost --unix socket-->
// here). Adapted from Stage 2's app/socketserver.go, now living where the
// heartbeats actually need to land so enforcement decisions don't depend on
// the app being alive.
type HeartbeatServer struct {
	mu       sync.Mutex
	conns    map[net.Conn]*bridgeConn
	listener net.Listener

	// everAlive tracks every browser name that has EVER sent a real
	// heartbeat, for the lifetime of this daemon process -- deliberately
	// never cleared, even after that bridge disconnects. This is what
	// enforce.go's missing-heartbeat rule gates on, in addition to
	// shared.HostRegistered: a browser can have a host manifest written
	// (auto-registered for every installed browser, not just ones the user
	// actually loaded the extension into) without ever having run the
	// extension. Only a browser that was actually heartbeating and then
	// went quiet is "missing" -- one that was never heartbeating in the
	// first place was never expected to be, and must never be targeted.
	everAlive map[string]bool

	lock   *LockManager
	config *ConfigStore
}

func NewHeartbeatServer(lock *LockManager, config *ConfigStore) *HeartbeatServer {
	return &HeartbeatServer{
		conns:     make(map[net.Conn]*bridgeConn),
		everAlive: make(map[string]bool),
		lock:      lock,
		config:    config,
	}
}

func (s *HeartbeatServer) Start() error {
	path, err := shared.HeartbeatSocketPath()
	if err != nil {
		return err
	}
	os.Remove(path) // stale socket from a previous run (no-op on Windows named pipes)
	ln, err := shared.ListenIPC(path)
	if err != nil {
		return err
	}
	s.listener = ln
	go s.acceptLoop()
	return nil
}

func (s *HeartbeatServer) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *HeartbeatServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return // listener closed
		}
		go s.handleConn(conn)
	}
}

func (s *HeartbeatServer) handleConn(conn net.Conn) {
	bc := &bridgeConn{conn: conn}
	s.mu.Lock()
	s.conns[conn] = bc
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
		conn.Close()
	}()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var msg shared.Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			log.Printf("sludge-exploder: bad heartbeat message: %v", err)
			continue
		}
		s.handleMessage(bc, msg)
	}
}

func (s *HeartbeatServer) handleMessage(bc *bridgeConn, msg shared.Message) {
	switch msg.Type {
	case shared.TypeBridgeHello:
		s.mu.Lock()
		bc.browser = msg.Browser
		s.mu.Unlock()

	case shared.TypeHello, shared.TypeHeartbeat:
		s.mu.Lock()
		bc.extID = msg.ExtID
		bc.version = msg.Version
		bc.configHash = msg.ConfigHash
		bc.lastHeartbeat = time.Now()
		if bc.browser != "" {
			s.everAlive[bc.browser] = true
		}
		s.mu.Unlock()
		s.reply(bc)

	case shared.TypeStatus:
		s.mu.Lock()
		bc.extID = msg.ExtID
		bc.version = msg.Version
		bc.configHash = msg.ConfigHash
		bc.rulesActive = msg.RulesActive
		bc.lastHeartbeat = time.Now()
		if bc.browser != "" {
			s.everAlive[bc.browser] = true
		}
		s.mu.Unlock()

	case shared.TypeSetConfigAck:
		s.mu.Lock()
		bc.configHash = msg.ConfigHash
		s.mu.Unlock()
	}
}

// reply answers a hello/heartbeat with the current lock state and the
// canonical config (see ConfigStore's comment for why this is always
// attached rather than gated on a hash match).
func (s *HeartbeatServer) reply(bc *bridgeConn) {
	state, until := s.lock.State()
	msg := shared.Message{
		Type:          shared.TypeStatus,
		LockState:     state,
		ConfigToApply: s.config.Current(),
	}
	if state == shared.LockStateLocked {
		msg.Until = until.Format(time.RFC3339)
	}
	s.send(bc, msg)
}

// PushSetConfig immediately broadcasts new rules to every connected bridge
// (the live-reload behavior from Stage 2). The heartbeat-reply fallback
// above covers bridges that were disconnected at the time of this push.
func (s *HeartbeatServer) PushSetConfig(rules []map[string]any) {
	s.broadcast(shared.Message{Type: shared.TypeSetConfig, Rules: rules})
}

func (s *HeartbeatServer) broadcast(msg shared.Message) {
	s.mu.Lock()
	conns := make([]*bridgeConn, 0, len(s.conns))
	for _, bc := range s.conns {
		conns = append(conns, bc)
	}
	s.mu.Unlock()

	for _, bc := range conns {
		s.send(bc, msg)
	}
}

func (s *HeartbeatServer) send(bc *bridgeConn, msg shared.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("sludge-exploder: marshaling heartbeat message: %v", err)
		return
	}
	data = append(data, '\n')

	bc.writeMu.Lock()
	defer bc.writeMu.Unlock()
	if _, err := bc.conn.Write(data); err != nil {
		log.Printf("sludge-exploder: writing to bridge: %v", err)
	}
}

// Snapshot returns the current known state of every connected bridge, for
// control-channel GetStatus responses.
func (s *HeartbeatServer) Snapshot() []shared.BrowserHeartbeatStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	out := make([]shared.BrowserHeartbeatStatus, 0, len(s.conns))
	for _, bc := range s.conns {
		out = append(out, shared.BrowserHeartbeatStatus{
			Browser:     bc.browser,
			Connected:   true,
			Alive:       isAlive(bc, now),
			ExtID:       bc.extID,
			Version:     bc.version,
			ConfigHash:  bc.configHash,
			RulesActive: bc.rulesActive,
		})
	}
	return out
}

// IsAlive reports whether the given process name's bridge is currently
// connected AND has heartbeated within heartbeatMissingAfter. Used by the
// enforcement tick to decide whether a controllable browser's extension has
// gone quiet. A browser with no bridge connected at all (extension
// disabled/removed, or the browser isn't running) is "not alive" too.
func (s *HeartbeatServer) IsAlive(processName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, bc := range s.conns {
		if bc.browser == processName {
			return isAlive(bc, now)
		}
	}
	return false
}

func isAlive(bc *bridgeConn, now time.Time) bool {
	return !bc.lastHeartbeat.IsZero() && now.Sub(bc.lastHeartbeat) < heartbeatMissingAfter
}

// EverAlive reports whether this browser has EVER sent a real heartbeat,
// for the lifetime of this daemon process. See the everAlive field comment
// above for why the missing-heartbeat enforcement rule requires this in
// addition to shared.HostRegistered.
func (s *HeartbeatServer) EverAlive(processName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.everAlive[processName]
}
