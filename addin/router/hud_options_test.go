// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// The in-canvas sketch input configuration round-trips over the wire (#2014).
func TestHUDOptionsRoundTrip(t *testing.T) {
	t.Parallel()
	r, s := New(opregistry.Default()), app.NewSession()
	var got wire.HeadsUpDisplayOptionsView
	call(t, r, s, wire.MethodApplicationGetHUDOptions, "{}", &got)
	if !got.Enabled || !got.DimensionInputEnabled || !got.CreateDimensionsOnValueInput {
		t.Fatalf("defaults = %+v, want the display, dimension input and value-input dimensions on", got)
	}

	var set wire.HeadsUpDisplayOptionsView
	call(t, r, s, wire.MethodApplicationSetHUDOptions,
		`{"enabled":true,"pointerInputEnabled":false,"pointerInputInCartesianCoordinates":true,`+
			`"dimensionInputEnabled":true,"dimensionInputInCartesianCoordinates":true,`+
			`"createDimensionsOnValueInput":false}`, &set)
	if set.PointerInputEnabled || !set.DimensionInputInCartesianCoordinates || set.CreateDimensionsOnValueInput {
		t.Errorf("set result = %+v, want pointer input off, cartesian dimensions on, value dimensions off", set)
	}

	var reread wire.HeadsUpDisplayOptionsView
	call(t, r, s, wire.MethodApplicationGetHUDOptions, "{}", &reread)
	if reread != set {
		t.Errorf("re-read = %+v, want the stored %+v", reread, set)
	}
}
