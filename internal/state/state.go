package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// State holds the persisted app settings.
type State struct {
	Enabled           bool   `json:"enabled"`
	PatchPath         string `json:"patch_path"`
	LastPatchedVersion string `json:"last_patched_version"`
	Autostart         bool   `json:"autostart"`
}

var (
	mu      sync.Mutex
	current State
	stateFile string
)

func init() {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".claude-telex")
	_ = os.MkdirAll(dir, 0o755)
	stateFile = filepath.Join(dir, "state.json")
	_ = Load()
}

// Load reads state from disk; uses defaults if file is missing.
func Load() error {
	mu.Lock()
	defer mu.Unlock()
	data, err := os.ReadFile(stateFile)
	if err != nil {
		// defaults
		current = State{Enabled: true}
		return nil
	}
	return json.Unmarshal(data, &current)
}

// Save writes state to disk.
func Save() error {
	mu.Lock()
	defer mu.Unlock()
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0o644)
}

// Get returns a copy of the current state (thread-safe).
func Get() State {
	mu.Lock()
	defer mu.Unlock()
	return current
}

// Update applies a mutation function and persists.
func Update(fn func(*State)) {
	mu.Lock()
	fn(&current)
	mu.Unlock()
	_ = Save()
}
