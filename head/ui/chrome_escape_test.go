//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestHandleEscapeRoutesByActiveInteraction checks each Escape branch: the transient nav
// interactions (Zoom Window, Constrained Orbit, SteeringWheels) are disarmed in priority order,
// and with none armed Esc forwards to the session's binding engine. handleEscape is pure (app
// calls only), so it needs no window.
func TestHandleEscapeRoutesByActiveInteraction(t *testing.T) {
	var none app.Modifier

	s := app.NewSession()
	s.ArmZoomWindow()
	handleEscape(s, none)
	if s.ZoomWindowArmed() {
		t.Error("Esc should disarm an armed Zoom Window")
	}

	s = app.NewSession()
	s.ToggleConstrainedOrbit()
	handleEscape(s, none)
	if s.ConstrainedOrbitActive() {
		t.Error("Esc should exit Constrained Orbit")
	}

	s = app.NewSession()
	s.ToggleSteeringWheel()
	handleEscape(s, none)
	if s.SteeringWheelActive() {
		t.Error("Esc should dismiss the SteeringWheels menu")
	}

	// Nothing armed: Esc forwards to the binding engine (cancels the active tool/selection).
	handleEscape(app.NewSession(), none) // default branch — must not panic
}
