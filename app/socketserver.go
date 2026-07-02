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

// bridgeConn tracks one connected nmhost relay (one browser's extension).
type bridgeConn struct {
	conn        net.Conn
	browser     string
	extID       string
	version     string
	configHash  string
	rulesActive int
	connectedAt time.Time
}

// BrowserStatus is the JSON shape returned to the webview UI for the setup
// helper and status panel.
type BrowserStatus struct {
	Browser     string `json:"browser"`
	Connected   bool   `json:"connected"`
	ExtID       string `json:"extId"`
	Version     string `json:"version"`
	ConfigHash  string `json:"configHash"`
	RulesActive int    `json:"rulesActive"`
}

// SocketServer accepts connections from nmhost relay processes over a local
// Unix socket and speaks shared.Message, newline-delimited, on that side.
type SocketServer struct {
	mu         sync.Mutex
	conns      map[net.Conn]*bridgeConn
	listener   net.Listener
	socketPath string
}

func NewSocketServer() *SocketServer {
	return &SocketServer{conns: make(map[net.Conn]*bridgeConn)}
}

func (s *SocketServer) Start() error {
	path, err := shared.SocketPath()
	if err != nil {
		return err
	}
	os.Remove(path) // stale socket from a previous run that didn't shut down cleanly
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	s.listener = ln
	s.socketPath = path
	go s.acceptLoop()
	return nil
}

func (s *SocketServer) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
	if s.socketPath != "" {
		os.Remove(s.socketPath)
	}
}

func (s *SocketServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return // listener closed
		}
		go s.handleConn(conn)
	}
}

func (s *SocketServer) handleConn(conn net.Conn) {
	bc := &bridgeConn{conn: conn, connectedAt: time.Now()}
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
			log.Printf("sludge-exploder: bad message from bridge: %v", err)
			continue
		}
		s.applyMessage(bc, msg)
	}
}

func (s *SocketServer) applyMessage(bc *bridgeConn, msg shared.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch msg.Type {
	case shared.TypeBridgeHello:
		bc.browser = msg.Browser
	case shared.TypeHello:
		bc.extID = msg.ExtID
		bc.version = msg.Version
	case shared.TypeStatus:
		bc.extID = msg.ExtID
		bc.version = msg.Version
		bc.configHash = msg.ConfigHash
		bc.rulesActive = msg.RulesActive
	case shared.TypeSetConfigAck:
		bc.configHash = msg.ConfigHash
	}
}

// PushSetConfig sends new rules to every connected bridge.
func (s *SocketServer) PushSetConfig(rules []map[string]any) {
	s.broadcast(shared.Message{Type: shared.TypeSetConfig, Rules: rules})
}

func (s *SocketServer) broadcast(msg shared.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("sludge-exploder: marshaling broadcast message: %v", err)
		return
	}
	data = append(data, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.conns {
		if _, err := conn.Write(data); err != nil {
			log.Printf("sludge-exploder: writing to bridge: %v", err)
		}
	}
}

// Snapshot returns the current known state of every connected bridge, for
// the setup helper and status panel.
func (s *SocketServer) Snapshot() []BrowserStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]BrowserStatus, 0, len(s.conns))
	for _, bc := range s.conns {
		out = append(out, BrowserStatus{
			Browser:     bc.browser,
			Connected:   true,
			ExtID:       bc.extID,
			Version:     bc.version,
			ConfigHash:  bc.configHash,
			RulesActive: bc.rulesActive,
		})
	}
	return out
}
