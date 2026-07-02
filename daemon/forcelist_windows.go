package main

import (
	"fmt"

	"golang.org/x/sys/windows/registry"

	"github.com/cnqso/sludge-exploder/shared"
)

// chromeWebStoreUpdateURL is what a real published extension's forcelist
// entry would point at. Writing this now is symbolic: per
// docs/ENFORCEMENT.md §4.4's "hard truth," Chrome only honors a
// forcelist entry for an extension it can actually fetch from an update
// URL, and ours isn't published there yet (Stage 5). This mechanism exists
// so the pieces are in place, not because it does anything observable yet.
const chromeWebStoreUpdateURL = "https://clients2.google.com/service/update2/crx"

// writeForceInstallPolicy writes a single ExtensionInstallForcelist entry
// for one Chrome-family browser, under HKEY_CURRENT_USER (no admin
// elevation, consistent with everything else this daemon writes).
func writeForceInstallPolicy(b shared.BrowserDef, extensionID string) error {
	if b.WindowsForcelistKey == "" {
		return fmt.Errorf("%s has no known forcelist policy key (Firefox has no equivalent mechanism)", b.Label)
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, b.WindowsForcelistKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening registry key %s: %w", b.WindowsForcelistKey, err)
	}
	defer key.Close()
	return key.SetStringValue("1", fmt.Sprintf("%s;%s", extensionID, chromeWebStoreUpdateURL))
}
