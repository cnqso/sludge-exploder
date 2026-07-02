// Command daemon is the Stage 3 "soft enforcement" daemon described in
// docs/DEVELOPMENT_PLAN.md and docs/ENFORCEMENT.md §4.3. It is NOT yet
// privileged or installed as a system service (that's Stage 4) -- it's a
// normal process you run by hand, and can be quit by hand. What it does
// have: a real lock state machine (persisted, survives being killed and
// relaunched), a heartbeat channel to the extension, and the ability to
// close a browser -- gated behind an explicit --enforce flag and, at
// runtime, the app's Enforcement safety toggle (see enforce.go).
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	enforceFlag := flag.Bool("enforce", false, "actually close browsers instead of only logging what would happen")
	uncontrolledFlag := flag.String("uncontrolled", "", `comma-separated process names to close whenever running during a lock (explicit opt-in test target only, e.g. --uncontrolled="Google Chrome" -- never defaulted to a real browser)`)
	gracePeriodFlag := flag.Duration("grace-period", 90*time.Second, "startup grace period per browser session, from when it launches -- missing/uncontrolled before this elapses is never closed; after it elapses, closed on the very first missed heartbeat (Cold Turkey-style: grace on launch, none for an established session)")
	flag.Parse()

	var uncontrolled []string
	if *uncontrolledFlag != "" {
		uncontrolled = strings.Split(*uncontrolledFlag, ",")
	}

	token, err := loadOrCreateToken()
	if err != nil {
		log.Fatalf("sludge-exploder: loading control token: %v", err)
	}

	lock := NewLockManager()
	config := NewConfigStore()
	enforcement := NewEnforcementState(*enforceFlag)

	hb := NewHeartbeatServer(lock, config)
	if err := hb.Start(); err != nil {
		log.Fatalf("sludge-exploder: starting heartbeat server: %v", err)
	}
	defer hb.Stop()

	risk := newRiskCache()

	control := NewControlServer(token, lock, hb, config, enforcement, risk)
	if err := control.Start(); err != nil {
		log.Fatalf("sludge-exploder: starting control server: %v", err)
	}
	defer control.Stop()

	go EnforcementLoop(lock, hb, newEnforcer(), enforcement, uncontrolled, 5*time.Second, *gracePeriodFlag, risk)

	if *enforceFlag {
		log.Printf("sludge-exploder daemon: enforcement ON at startup -- will actually close browsers")
	} else {
		log.Printf("sludge-exploder daemon: enforcement OFF (log-only) -- pass --enforce to actually close browsers")
	}
	log.Printf("sludge-exploder daemon: grace period %s before closing anything", gracePeriodFlag.String())
	if len(uncontrolled) > 0 {
		log.Printf("sludge-exploder daemon: uncontrolled browsers: %v", uncontrolled)
	}
	if state, until := lock.State(); state == "LOCKED" {
		log.Printf("sludge-exploder daemon: resuming a lock persisted from a previous run, until %s", until.Format(time.RFC3339))
	}

	// Block until interrupted. Matches Stage 3 scope: no re-spawn-on-kill,
	// no service-manager integration yet -- that's Stage 4.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Println("sludge-exploder daemon: shutting down")
}
