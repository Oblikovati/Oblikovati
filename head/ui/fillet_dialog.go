//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Fillet flow in the head: while the Fillet tool runs, a modeless options window shows
// the picked-edge count and the blend radius (database units), then OK/Cancel.
var filletUI = struct {
	radius float32
	open   bool
}{radius: 1}

// drawFilletDialog shows the fillet options window while the Fillet tool is active.
func drawFilletDialog(s *app.Session) {
	f := s.ActiveFillet()
	if f == nil {
		filletUI.open = false
		return
	}
	if !filletUI.open {
		filletUI.radius = float32(f.Radius())
		filletUI.open = true
	}
	native.SetNextWindowSize(300, 160)
	if native.Begin("Fillet") {
		native.Text("Edges: " + strconv.Itoa(len(f.Edges())) + " (click convex edges to round)")
		native.Text("Radius (" + s.LengthUnitName() + ")")
		native.InputFloat("##fillet-radius", &filletUI.radius)
		f.SetRadius(float64(filletUI.radius))
		drawCommitCancelButtons(s, f.CanCommit())
	}
	native.End()
}
