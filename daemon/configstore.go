package main

import (
	"encoding/json"
	"fmt"
	"sync"
)

// ConfigStore holds the canonical SLUDGE_CONFIG rules the daemon pushes to
// every connected bridge: immediately on SetConfig (see heartbeat.go's
// PushSetConfig), and as a fallback attached to every heartbeat reply so a
// browser that was closed during a push still self-heals on reconnect.
//
// The fallback is unconditional (always attached, not gated on a hash
// match) deliberately: matching a hash computed by extension/hash.js
// against one computed here would require Go's json.Marshal and JS's
// JSON.stringify to agree on key ordering byte-for-byte, which is fragile
// to depend on for correctness. Configs are tiny and heartbeats are cheap,
// so "just always send it" is both simpler and more robust.
type ConfigStore struct {
	mu    sync.Mutex
	rules []map[string]any
}

func NewConfigStore() *ConfigStore {
	return &ConfigStore{}
}

func (c *ConfigStore) Set(rules []map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules = rules
}

func (c *ConfigStore) Current() []map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rules
}

// Hash is FNV-1a over the marshaled rules, matching extension/hash.js's
// algorithm (though not necessarily its exact byte output -- see the
// type-level comment). Used only for display/debugging in GetStatus
// responses, never for a correctness-critical comparison.
func (c *ConfigStore) Hash() string {
	rules := c.Current()
	data, err := json.Marshal(rules)
	if err != nil {
		return ""
	}
	var hash uint32 = 0x811c9dc5
	for _, b := range data {
		hash ^= uint32(b)
		hash *= 0x01000193
	}
	return fmt.Sprintf("%x", hash)
}
