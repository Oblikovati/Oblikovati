//go:build linux || darwin

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"syscall"
	"testing"
	"time"
)

// TestMainStopsOnSignal is the entrypoint's end-to-end shutdown proof: main() builds the
// session, installs the host, scans for add-ins, and enters the drain loop, and a real
// SIGTERM to this process unwinds it cleanly (runDrainLoop's os/signal wait returns and
// main falls through). Unix-only — syscall.Kill has no Windows counterpart, and the head
// coverage job (Linux) is where this path is measured; the loop body itself is also unit
// tested platform-independently via drainUntil/drainTick.
func TestMainStopsOnSignal(t *testing.T) {
	done := make(chan struct{})
	go func() { main(); close(done) }()

	// runDrainLoop's signal.Notify is reached within main()'s first few statements (a
	// session build + router construction, all sub-100ms); this wait leaves ample room so
	// the SIGTERM lands on its channel, not the default terminate action.
	time.Sleep(500 * time.Millisecond)
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("raise SIGTERM: %v", err)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("main did not return after SIGTERM")
	}
}
