package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/cnqso/sludge-exploder/shared"
)

// LockManager owns the UNLOCKED -> LOCKED(until=T) -> UNLOCKED state
// machine, persisted to shared.LockStatePath() so it survives the daemon
// being killed and relaunched (including across a reboot, if relaunched
// afterward -- Stage 3 doesn't auto-start the daemon on login, that's
// Stage 4's job).
type LockManager struct {
	mu    sync.Mutex
	until time.Time // zero value means UNLOCKED
}

type lockStateFile struct {
	Until time.Time `json:"until"`
}

func NewLockManager() *LockManager {
	lm := &LockManager{}
	lm.load()
	return lm
}

func (lm *LockManager) load() {
	path, err := shared.LockStatePath()
	if err != nil {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var f lockStateFile
	if err := json.Unmarshal(data, &f); err != nil {
		log.Printf("sludge-exploder: parsing persisted lock state: %v", err)
		return
	}
	lm.mu.Lock()
	lm.until = f.Until
	lm.mu.Unlock()
}

func (lm *LockManager) persist() {
	path, err := shared.LockStatePath()
	if err != nil {
		log.Printf("sludge-exploder: resolving lock state path: %v", err)
		return
	}
	lm.mu.Lock()
	data, err := json.Marshal(lockStateFile{Until: lm.until})
	lm.mu.Unlock()
	if err != nil {
		return
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		log.Printf("sludge-exploder: persisting lock state: %v", err)
	}
}

// State returns the current lock state and, if LOCKED, when it expires.
func (lm *LockManager) State() (state string, until time.Time) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.until.After(time.Now()) {
		return shared.LockStateLocked, lm.until
	}
	return shared.LockStateUnlocked, time.Time{}
}

// StartLock sets a new lock. Always allowed -- extending an existing lock
// (rather than refusing) is a deliberate choice: there's no safety reason
// to block "lock for even longer."
func (lm *LockManager) StartLock(duration time.Duration) time.Time {
	lm.mu.Lock()
	lm.until = time.Now().Add(duration)
	until := lm.until
	lm.mu.Unlock()
	lm.persist()
	return until
}

// StopLock unlocks immediately, but refuses while a lock is still active.
// That refusal is the product, not a limitation -- see
// docs/DEVELOPMENT_PLAN.md Stage 3 DoD: "StopLock is refused by the daemon
// while LOCKED."
func (lm *LockManager) StopLock() error {
	lm.mu.Lock()
	if lm.until.After(time.Now()) {
		remaining := time.Until(lm.until).Round(time.Second)
		lm.mu.Unlock()
		return fmt.Errorf("cannot stop an active lock (%s remaining)", remaining)
	}
	lm.until = time.Time{}
	lm.mu.Unlock()
	lm.persist()
	return nil
}
