package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func pausePath() string {
	return filepath.Join(GritDir(), "pause")
}

func IsPaused() bool {
	data, err := os.ReadFile(pausePath())
	if err != nil {
		return false
	}
	s := strings.TrimSpace(string(data))
	if s == "disabled" {
		return true
	}
	until, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return false
	}
	return time.Now().Before(until)
}

func Pause(d time.Duration) error {
	until := time.Now().Add(d).Format(time.RFC3339)
	return os.WriteFile(pausePath(), []byte(until), 0644)
}

func Disable() error {
	return os.WriteFile(pausePath(), []byte("disabled"), 0644)
}

func Resume() error {
	err := os.Remove(pausePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// PauseStatus returns a human-readable description of the current pause state,
// or an empty string if grit is active.
func PauseStatus() string {
	data, err := os.ReadFile(pausePath())
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(data))
	if s == "disabled" {
		return "disabled indefinitely"
	}
	until, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return ""
	}
	if time.Now().Before(until) {
		remaining := time.Until(until).Round(time.Minute)
		return "snoozed for " + remaining.String()
	}
	return ""
}
