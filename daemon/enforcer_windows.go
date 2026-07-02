package main

import (
	"fmt"
	"os/exec"
	"strings"
)

type windowsEnforcer struct{}

func newEnforcer() Enforcer {
	return windowsEnforcer{}
}

func (windowsEnforcer) IsRunning(processName string) bool {
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName), "/NH", "/FO", "CSV").Output()
	if err != nil {
		return false
	}
	// tasklist prints a CSV row per match (first field quoted image name)
	// when found, or an "INFO: No tasks..." line when not -- checking for
	// the quoted process name is more robust than trying to distinguish
	// those message formats, which can vary by Windows locale.
	return strings.Contains(string(out), fmt.Sprintf(`"%s"`, processName))
}

func (windowsEnforcer) KillBrowser(processName string) error {
	if err := exec.Command("taskkill", "/IM", processName, "/F").Run(); err != nil {
		return fmt.Errorf("taskkill /IM %q /F: %w", processName, err)
	}
	return nil
}
