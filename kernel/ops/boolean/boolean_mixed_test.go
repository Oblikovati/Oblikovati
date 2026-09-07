// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestBooleanMixedPassesTheBossThrough: a bossed block minus a notch far from the boss takes the
// EXACT per-face-dispatch boolean (ADR-0058) — the boss cylinder passes through analytically and
// nothing degrades. Before the dispatch this class fell to the mesh-arrangement reconstruction; that
// engine is gone (ADR-0061 stage 7), so the assertion is now that no degradation is recorded at all.
func TestBooleanMixedPassesTheBossThrough(t *testing.T) {
	t.Parallel()
	block, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	cyl, _ := brep.SolidCylinder(math.P3(5, 5, 10), math.V3(0, 0, 1), 2, 3)
	bossed, err := brep.Boolean(brep.Union, block, cyl)
	if err != nil {
		t.Fatalf("boss fixture: %v", err)
	}
	notch, _ := brep.SolidBlock(math.P3(-1, 4, 1), math.P3(2, 6, 3), "notch")

	rec := &diag.Recorder{}
	res, err := BooleanWithDiagnostics(Cut, bossed, notch, rec)
	if err != nil || res == nil {
		t.Fatalf("mixed cut failed: %v", err)
	}
	if rec.Has(CodeBooleanNoExactCurvedPath) || rec.Has(CodeBooleanAnalyticFaceted) {
		t.Errorf("the mixed cut degraded; want the exact per-face-dispatch boolean: %v", rec.Records())
	}
	cyls := 0
	for _, f := range res.Faces() {
		if _, isCyl := f.Geometry().(geom.Cylinder); isCyl {
			cyls++
		}
	}
	if cyls != 1 {
		t.Errorf("boss wall did not survive analytically: %d cylinder faces, want 1", cyls)
	}
	want := 1000 + stdmath.Pi*4*3 - 8
	if got := analyticVolumeOf(t, res); stdmath.Abs(got-want) > 0.5 {
		t.Errorf("mixed cut volume = %g, want %g", got, want)
	}
}

// analyticVolumeOf reads the body's volume through the mass-properties pipeline.
func analyticVolumeOf(t *testing.T, b *topo.Body) float64 {
	t.Helper()
	return query.BodyGeometryProperties(b, DefaultQuality()).Volume
}

// TestBooleanEmbeddedCavityExactVolume is the regression for the DrillThroughHole span-gate misfire:
// cutting a cylinder EMBEDDED inside a block (not spanning it) used to be mis-recognized as a full
// through-hole (silently removing π·r²·H instead of π·r²·h). That recipe is deleted (ADR-0061 stage 4);
// the per-face dispatch bounds the bore by the tool's OWN band, so it cuts the exact cavity.
func TestBooleanEmbeddedCavityExactVolume(t *testing.T) {
	t.Parallel()
	block, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	tool, _ := brep.SolidCylinder(math.P3(5, 5, 3), math.V3(0, 0, 1), 1, 4)
	res, err := Boolean(Cut, block, tool)
	if err != nil || res == nil {
		t.Fatalf("embedded cavity cut failed: %v", err)
	}
	want := 1000 - stdmath.Pi*4 // NOT 1000 − π·10 (the old through-hole misfire)
	if got := analyticVolumeOf(t, res); stdmath.Abs(got-want) > 0.5 {
		t.Errorf("embedded cavity volume = %g, want %g", got, want)
	}
	if !Validate(res).ValidSolid() {
		t.Error("embedded cavity result is not a valid solid")
	}
}

// TestBooleanRoutesMixedDeclineToFallback: when the per-face dispatch declines (here a boundaryless
// sphere face, which carries no boundary point to classify the face as a whole), ops.Boolean routes on
// the named sentinel to the curved/CSG paths and still returns the right solid — a decline is a change
// of route, never a failure the caller sees.
func TestBooleanRoutesMixedDeclineToFallback(t *testing.T) {
	t.Parallel()
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	ball, err := brep.SolidSphere(math.P3(30, 30, 30), 2, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	res, err := Boolean(Join, block, ball)
	if err != nil {
		t.Fatalf("Boolean(Join) after the mixed decline: %v", err)
	}
	if got := len(res.Shells()); got != 2 {
		t.Errorf("union of two disjoint solids has %d shells, want 2", got)
	}
	if r := Validate(res); !r.Valid {
		t.Errorf("union after the mixed decline is invalid: %v", r.Issues)
	}
}
