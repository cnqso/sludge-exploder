package main

import (
	"crypto/rand"
	"encoding/hex"
	"os"

	"github.com/cnqso/sludge-exploder/shared"
)

// loadOrCreateToken returns the control-socket auth token, generating and
// persisting a new random one on first run. File permissions (0600) are
// what make this a meaningful secret in Stage 3: only the local user both
// the app and daemon run as can read it. Stage 4's real privilege
// separation carries the same mechanism forward once the daemon runs as a
// separate system account.
func loadOrCreateToken() (string, error) {
	path, err := shared.ControlTokenPath()
	if err != nil {
		return "", err
	}
	if data, err := os.ReadFile(path); err == nil {
		return string(data), nil
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		return "", err
	}
	return token, nil
}
