// SPDX-License-Identifier: GPL-2.0-only

package windowstate

import (
	"runtime"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("overrides $HOME to redirect the config dir; Unix only")
	}
	t.Setenv("HOME", t.TempDir())

	want := State{X: 120, Y: 64, Width: 1600, Height: 1000, Maximized: true}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, ok := Load()
	if !ok {
		t.Fatal("Load: not found after Save")
	}
	if got != want {
		t.Errorf("round-trip = %+v, want %+v", got, want)
	}
}

func TestLoadMissingIsNotOK(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Linux only")
	}
	t.Setenv("HOME", t.TempDir())
	if _, ok := Load(); ok {
		t.Error("Load on empty config should report not-found")
	}
}

func TestInvalidStateIgnored(t *testing.T) {
	if !(State{Width: 0, Height: 0}).Valid() == false {
		t.Error("zero-size state should be invalid")
	}
	if err := Save(State{}); err != nil {
		t.Errorf("Save of invalid state should be a no-op, got %v", err)
	}
}
