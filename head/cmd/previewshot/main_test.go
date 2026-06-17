//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/renderer"
)

// previewSession builds the driver's base box and starts a pending extrude of the given
// operation, returning the session whose ToolPreview should carry the live preview ghost.
func previewSession(t *testing.T, op string) *app.Session {
	t.Helper()
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	startPendingExtrude(s, buildBaseBox(s), op)
	return s
}

// translucentTriangles returns the translucent triangle items of a tool preview — the feature
// ghost, distinct from any wireframe.
func translucentTriangles(items []renderer.DrawItem) []renderer.DrawItem {
	var out []renderer.DrawItem
	for _, it := range items {
		if it.Primitive == renderer.Triangles && it.Opacity > 0 && it.Opacity < 1 {
			out = append(out, it)
		}
	}
	return out
}

// TestPendingExtrudePreviewColors checks the driver's pending extrude shows a translucent
// ghost that is green-dominant for a material-adding join and red-dominant for a cut — the
// behaviour the live PNGs capture (run() opens a real window and is exercised manually).
func TestPendingExtrudePreviewColors(t *testing.T) {
	for _, tc := range []struct {
		op         string
		wantGreen  bool
		moreOfChan int // index of the channel that must dominate (0=R, 1=G)
		lessChan   int
	}{
		{op: "join", wantGreen: true, moreOfChan: 1, lessChan: 0},
		{op: "cut", wantGreen: false, moreOfChan: 0, lessChan: 1},
	} {
		t.Run(tc.op, func(t *testing.T) {
			s := previewSession(t, tc.op)
			ghost := translucentTriangles(s.ToolPreview())
			if len(ghost) == 0 {
				t.Fatalf("%s: no translucent preview items", tc.op)
			}
			c := ghost[0].Color
			if c[tc.moreOfChan] <= c[tc.lessChan] {
				t.Errorf("%s preview color = %v, want channel %d to dominate", tc.op, c, tc.moreOfChan)
			}
		})
	}
}
