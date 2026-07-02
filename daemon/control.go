package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/cnqso/sludge-exploder/shared"
)

// ControlServer is the app's side of the daemon: unlike the heartbeat
// server (which nmhost relays into), the app is a normal long-running
// process and dials this socket directly -- no relay needed. Each
// connection is a single authenticated request/response, not a persistent
// session.
type ControlServer struct {
	token       string
	lock        *LockManager
	hb          *HeartbeatServer
	config      *ConfigStore
	enforcement *EnforcementState
	risk        *riskCache
	listener    net.Listener
}

func NewControlServer(token string, lock *LockManager, hb *HeartbeatServer, config *ConfigStore, enforcement *EnforcementState, risk *riskCache) *ControlServer {
	return &ControlServer{token: token, lock: lock, hb: hb, config: config, enforcement: enforcement, risk: risk}
}

func (s *ControlServer) Start() error {
	path, err := shared.ControlSocketPath()
	if err != nil {
		return err
	}
	os.Remove(path) // stale socket from a previous run (no-op on Windows named pipes)
	ln, err := shared.ListenIPC(path)
	if err != nil {
		return err
	}
	os.Chmod(path, 0o600) // defense in depth alongside the token (no-op on Windows; named pipes get their own ACL via go-winio's default security descriptor)
	s.listener = ln
	go s.acceptLoop()
	return nil
}

func (s *ControlServer) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *ControlServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return // listener closed
		}
		go s.handleConn(conn)
	}
}

func (s *ControlServer) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	if !scanner.Scan() {
		return
	}

	var msg shared.Message
	if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
		s.respond(conn, shared.Message{Type: shared.TypeError, Error: "bad request"})
		return
	}
	if msg.Token != s.token {
		s.respond(conn, shared.Message{Type: shared.TypeError, Error: "unauthorized"})
		return
	}

	s.respond(conn, s.dispatch(msg))
}

func (s *ControlServer) respond(conn net.Conn, msg shared.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	conn.Write(append(data, '\n'))
}

func (s *ControlServer) dispatch(msg shared.Message) shared.Message {
	switch msg.Type {
	case shared.TypeStartLock:
		duration := time.Duration(msg.DurationSeconds) * time.Second
		if duration <= 0 {
			return shared.Message{Type: shared.TypeError, Error: "durationSeconds must be positive"}
		}
		until := s.lock.StartLock(duration)
		return s.statusMessage(shared.LockStateLocked, until)

	case shared.TypeStopLock:
		if err := s.lock.StopLock(); err != nil {
			return shared.Message{Type: shared.TypeError, Error: err.Error()}
		}
		return s.statusMessage(shared.LockStateUnlocked, time.Time{})

	case shared.TypeSetConfig:
		s.config.Set(msg.Rules)
		s.hb.PushSetConfig(msg.Rules)
		state, until := s.lock.State()
		return s.statusMessage(state, until)

	case shared.TypeSetEnforcement:
		s.enforcement.Set(msg.Enabled)
		state, until := s.lock.State()
		return s.statusMessage(state, until)

	case shared.TypeGetStatus:
		state, until := s.lock.State()
		return s.statusMessage(state, until)

	default:
		return shared.Message{Type: shared.TypeError, Error: fmt.Sprintf("unknown command %q", msg.Type)}
	}
}

func (s *ControlServer) statusMessage(state string, until time.Time) shared.Message {
	m := shared.Message{
		Type:        shared.TypeLockStatus,
		LockState:   state,
		Enforcement: s.enforcement.Enabled(),
		Browsers:    s.hb.Snapshot(),
		ConfigHash:  s.config.Hash(),
		RulesActive: len(s.config.Current()),
		Risk:        s.riskSnapshot(),
	}
	if state == shared.LockStateLocked {
		m.Until = until.Format(time.RFC3339)
	}
	return m
}

// riskSnapshot reads the enforcement tick's cached risk state (no extra
// IsRunning/pgrep calls triggered by an app poll) for every known browser
// currently at risk, so the app can warn the user while there's still time
// to react.
func (s *ControlServer) riskSnapshot() []shared.BrowserRisk {
	var out []shared.BrowserRisk
	for _, b := range shared.KnownBrowsers() {
		info := s.risk.get(b.ProcessName)
		if !info.AtRisk {
			continue
		}
		out = append(out, shared.BrowserRisk{
			Browser:               b.ProcessName,
			Label:                 b.Label,
			GraceRemainingSeconds: int(info.GraceRemaining.Round(time.Second).Seconds()),
		})
	}
	return out
}
