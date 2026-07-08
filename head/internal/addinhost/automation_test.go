//go:build linux || darwin || windows

// SPDX-License-Identifier: GPL-2.0-only

package addinhost

import "testing"

// TestAutomationExportRoundTrips loads the echo fixture and drives its OPTIONAL
// ObkAddInAutomation export end-to-end: dlsym resolution, the C call, and the
// cross-runtime buffer copy + ObkFree release (M05-F01 #252). No host callback is
// involved — automation is a direct host→add-in call.
func TestAutomationExportRoundTrips(t *testing.T) {
	so := buildFixture(t, "echoaddin")
	lib, err := Open(so)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if err := lib.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	if !lib.HasAutomation() {
		t.Fatal("HasAutomation = false, want the fixture's ObkAddInAutomation resolved")
	}
	out, err := lib.CallAutomation("solve", []byte("abc"))
	if err != nil {
		t.Fatalf("CallAutomation: %v", err)
	}
	if string(out) != `{"method":"solve","payload":"abc"}` {
		t.Errorf("automation reply = %s, want the fixture's echo", out)
	}
}

// TestAutomationAbsentIsReported loads the UI fixture (which has no automation
// export) and checks the probe reports false and the call fails with a clear error
// — the lenient-resolution contract of the header.
func TestAutomationAbsentIsReported(t *testing.T) {
	so := buildFixture(t, "uiaddin")
	lib, err := Open(so)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if err := lib.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	if lib.HasAutomation() {
		t.Fatal("HasAutomation = true for a fixture without the export")
	}
	if _, err := lib.CallAutomation("x", nil); err == nil {
		t.Fatal("CallAutomation should fail when the export is absent")
	}
}
