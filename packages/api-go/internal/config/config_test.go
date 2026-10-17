package config

import (
	"testing"
	"time"
)

func TestGetDurationEnv(t *testing.T) {
	const key = "NOTIFY_ROOM_COMMENT_WINDOW"
	def := 30 * time.Minute
	for raw, want := range map[string]time.Duration{
		"":     def, // unset
		"10m":  10 * time.Minute,
		"1h":   time.Hour,
		"30":   def, // no unit
		"soon": def,
		"0s":   def,
		"-5m":  def,
	} {
		t.Setenv(key, raw)
		if got := getDurationEnv(key, def); got != want {
			t.Errorf("%q: got %s, want %s", raw, got, want)
		}
	}
}
