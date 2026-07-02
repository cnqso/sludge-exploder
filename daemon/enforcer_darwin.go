package main

import (
	"fmt"
	"os/exec"
)

type darwinEnforcer struct{}

func newEnforcer() Enforcer {
	return darwinEnforcer{}
}

func (darwinEnforcer) IsRunning(processName string) bool {
	return exec.Command("pgrep", "-x", processName).Run() == nil
}

func (darwinEnforcer) KillBrowser(processName string) error {
	if err := exec.Command("pkill", "-x", processName).Run(); err != nil {
		return fmt.Errorf("pkill -x %q: %w", processName, err)
	}
	return nil
}
