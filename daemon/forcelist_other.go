//go:build !windows

package main

import (
	"fmt"

	"github.com/cnqso/sludge-exploder/shared"
)

// Force-install policy writing is Windows-only for now (see
// forcelist_windows.go); a macOS .mobileconfig equivalent is a documented
// future piece (docs/ENFORCEMENT.md §4.4), not built yet.
func writeForceInstallPolicy(b shared.BrowserDef, extensionID string) error {
	return fmt.Errorf("force-install policy isn't implemented on this platform yet")
}
