package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Prefs is the user's saved preferences: which catalog parts are enabled,
// custom domains, the lock duration, and the last lock intent (Stage 2's
// Lock button doesn't enforce anything -- it just records this).
type Prefs struct {
	// Selections[appID][partID] = enabled.
	Selections      map[string]map[string]bool `json:"selections"`
	CustomDomains   []CustomDomain             `json:"customDomains"`
	DurationMinutes int                        `json:"durationMinutes"`
	LastLockIntent  *LockIntent                `json:"lastLockIntent,omitempty"`
}

type CustomDomain struct {
	Domain    string   `json:"domain"`
	Selectors []string `json:"selectors"`
}

type LockIntent struct {
	DurationMinutes int    `json:"durationMinutes"`
	Summary         string `json:"summary"`
	ConfirmedAt     string `json:"confirmedAt"` // RFC3339
}

func defaultPrefs() Prefs {
	return Prefs{Selections: map[string]map[string]bool{}, DurationMinutes: 60}
}

func prefsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(dir, "SludgeExploder")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "prefs.json"), nil
}

func loadPrefs() (Prefs, error) {
	path, err := prefsPath()
	if err != nil {
		return defaultPrefs(), err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultPrefs(), nil
		}
		return defaultPrefs(), err
	}
	p := defaultPrefs()
	if err := json.Unmarshal(data, &p); err != nil {
		return defaultPrefs(), err
	}
	if p.Selections == nil {
		p.Selections = map[string]map[string]bool{}
	}
	return p, nil
}

func savePrefsToDisk(p Prefs) error {
	path, err := prefsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
