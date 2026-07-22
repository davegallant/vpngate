package daemon

import (
	"encoding/json"
	"os"
	"time"
)

// State is the on-disk record of a running daemon, written by the
// supervisor once OpenVPN reports a successful connection and read by
// `status`/`disconnect` in separate process invocations.
type State struct {
	PID         int       `json:"pid"`
	ControlAddr string    `json:"control_addr"`
	HostName    string    `json:"hostname"`
	IPAddr      string    `json:"ip_addr"`
	CountryLong string    `json:"country"`
	StartedAt   time.Time `json:"started_at"`
}

// Save writes state to StatePath(), creating Dir() if needed.
func Save(state State) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(StatePath(), data, 0o600)
}

// Load reads and parses StatePath(). Callers should check
// os.IsNotExist(err) to distinguish "no daemon running" from a real
// error.
func Load() (State, error) {
	var state State
	data, err := os.ReadFile(StatePath())
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(data, &state)
	return state, err
}

// Remove deletes StatePath(). It is not an error if the file is already
// gone.
func Remove() error {
	err := os.Remove(StatePath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
