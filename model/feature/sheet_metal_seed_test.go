// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/param"
)

// seedSheetMetalSheet builds a square sheet-metal wall (side cm, 2 mm thick) and returns the
// engine and a top +X boundary edge to fold from — the common fixture the bend-family
// feature tests (flange/hem/bend) share so their setup is not duplicated. extraParams adds
// further parameters (e.g. BendRadius) the test needs.
func seedSheetMetalSheet(t *testing.T, side float64, extraParams map[string]string) (*PartFeatures, *topo.Edge) {
	t.Helper()
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	for name, expr := range extraParams {
		mustParam(t, ps, name, expr)
	}
	fs := NewPartFeatures(ps, nil)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(side), ProfileIndex: 0, Operation: ops.NewBody})
	fs.Recompute()
	return fs, topEdgeAlongX(t, fs.Result()[0])
}

// smSolidVolume is a sheet-metal body's volume at a fine chord tolerance — the measure the
// roll/loft geometry tests compare against an analytic value.
func smSolidVolume(body *topo.Body) float64 {
	return ops.BodyGeometryProperties(body, ops.Quality{ChordTolerance: 1e-3}).Volume
}
