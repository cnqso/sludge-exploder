package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cnqso/sludge-exploder/shared"
)

// DaemonClient is a thin client for the daemon's control socket. Unlike
// nmhost's relay to the heartbeat socket, the app is a normal long-running
// process and can dial the daemon directly -- no relay needed. Each call
// opens a fresh connection, sends one authenticated request, reads one
// response, and closes; there's no persistent session to manage.
type DaemonClient struct {
	token string
}

// NewDaemonClient loads the control-socket auth token from disk. Returns an
// error if the daemon has never run (the token file doesn't exist yet) --
// callers should treat that as "daemon not available," not fatal.
func NewDaemonClient() (*DaemonClient, error) {
	path, err := shared.ControlTokenPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading control token (is the daemon running?): %w", err)
	}
	return &DaemonClient{token: string(data)}, nil
}

func (c *DaemonClient) send(msg shared.Message) (shared.Message, error) {
	path, err := shared.ControlSocketPath()
	if err != nil {
		return shared.Message{}, err
	}

	conn, err := shared.DialIPC(path, 2*time.Second)
	if err != nil {
		return shared.Message{}, fmt.Errorf("connecting to daemon (is it running?): %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	msg.Token = c.token
	data, err := json.Marshal(msg)
	if err != nil {
		return shared.Message{}, err
	}
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return shared.Message{}, err
	}

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	if !scanner.Scan() {
		return shared.Message{}, fmt.Errorf("no response from daemon")
	}

	var resp shared.Message
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return shared.Message{}, err
	}
	if resp.Type == shared.TypeError {
		return shared.Message{}, fmt.Errorf("daemon: %s", resp.Error)
	}
	return resp, nil
}

func (c *DaemonClient) GetStatus() (shared.Message, error) {
	return c.send(shared.Message{Type: shared.TypeGetStatus})
}

func (c *DaemonClient) SetConfig(rules []map[string]any) (shared.Message, error) {
	return c.send(shared.Message{Type: shared.TypeSetConfig, Rules: rules})
}

func (c *DaemonClient) StartLock(duration time.Duration) (shared.Message, error) {
	return c.send(shared.Message{Type: shared.TypeStartLock, DurationSeconds: int(duration.Seconds())})
}

// StopLock is expected to fail (an explicit refusal) whenever a lock is
// still active -- that's the product, not a bug. See daemon/lock.go.
func (c *DaemonClient) StopLock() (shared.Message, error) {
	return c.send(shared.Message{Type: shared.TypeStopLock})
}

func (c *DaemonClient) SetEnforcement(enabled bool) (shared.Message, error) {
	return c.send(shared.Message{Type: shared.TypeSetEnforcement, Enabled: enabled})
}
