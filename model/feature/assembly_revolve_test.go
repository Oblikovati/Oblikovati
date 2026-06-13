// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// fullTurn is the revolve-angle closure for a complete 360° revolution.
func fullTurn() float64 { return 2 * stdmath.Pi }

// TestAssemblyRevolveToolVolume gates the revolved tool against the analytic value: the
// square x∈[2,4], y∈[0,2] spun a full turn about the Y axis is a washer of volume 24π.
func TestAssemblyRevolveToolVolume(t *testing.T) {
	f := NewAssemblyRevolveFeature(offsetSquareSketch(2, 2), 0, yAxis(), ops.Cut, fullTurn)
	if f.Kind() != "assemblyRevolve" {
		t.Errorf("kind = %q, want assemblyRevolve", f.Kind())
	}
	tool, err := f.buildTool()
	if err != nil {
		t.Fatalf("buildTool: %v", err)
	}
	want := stdmath.Pi * (4*4 - 2*2) * 2 // 24π washer
	if got := bodyVolume(tool); relErr(got, want) > 0.01 {
		t.Errorf("revolved tool volume = %g, want ≈%g (24π)", got, want)
	}
}

// enclosingBlock builds the box [-2,2]×[0,1]×[-2,2] (volume 16) that strictly contains the
// unit cylinder turned in the cut test — no face tangency — so the boolean removes the tool's
// whole volume cleanly.
func enclosingBlock(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(math.P3(-2, 0, -2), math.P3(2, 1, 2), "target")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}

// TestAssemblyRevolveCutsEveryBody is the assembly-context cut over several bodies: a
// full-turn revolve of the square x∈[0,1], y∈[0,1] about the Y axis is a cylinder (r≈1,
// y∈[0,1], volume ≈π by the analytic gate), strictly inside each enclosing block — so the
// boolean removes exactly the tool volume from every participant (16 − toolVol).
func TestAssemblyRevolveCutsEveryBody(t *testing.T) {
	f := NewAssemblyRevolveFeature(offsetSquareSketch(0, 1), 0, yAxis(), ops.Cut, fullTurn)
	tool, err := f.buildTool()
	if err != nil {
		t.Fatalf("buildTool: %v", err)
	}
	if toolVol := bodyVolume(tool); relErr(toolVol, stdmath.Pi) > 0.01 {
		t.Fatalf("turned cylinder tool = %g, want ≈%g (π)", toolVol, stdmath.Pi)
	}

	out, err := f.Recompute(Input{Bodies: []*topo.Body{enclosingBlock(t), enclosingBlock(t)}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 2 {
		t.Fatalf("result bodies = %d, want 2", len(out.Bodies))
	}
	want := 16.0 - bodyVolume(tool) // each block keeps its volume minus the fully-contained tool
	for i, b := range out.Bodies {
		if got := bodyVolume(b); relErr(got, want) > 0.005 {
			t.Errorf("body %d revolve-cut volume = %g, want ≈%g (block minus the turned cylinder)", i, got, want)
		}
	}
}

// TestAssemblyRevolveResolvesSketchCenterline: with no explicit axis the feature spins about
// the sketch's single centerline (Inventor's common flow), producing the same washer.
func TestAssemblyRevolveResolvesSketchCenterline(t *testing.T) {
	sk := offsetSquareSketch(2, 2)
	cl := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2)) // vertical centerline = Y axis
	cl.SetCenterline(true)

	tool, err := NewAssemblyRevolveFeature(sk, 0, nil, ops.Cut, fullTurn).buildTool()
	if err != nil {
		t.Fatalf("buildTool about centerline: %v", err)
	}
	want := stdmath.Pi * (4*4 - 2*2) * 2
	if got := bodyVolume(tool); relErr(got, want) > 0.01 {
		t.Errorf("centerline-revolved tool = %g, want ≈%g (24π)", got, want)
	}
}

// TestAssemblyRevolveRejectsBadAngle: a non-positive angle is a lost input reported as an
// error rather than a degenerate solid.
func TestAssemblyRevolveRejectsBadAngle(t *testing.T) {
	f := NewAssemblyRevolveFeature(offsetSquareSketch(2, 2), 0, yAxis(), ops.Cut, func() float64 { return 0 })
	if _, err := f.Recompute(Input{Bodies: []*topo.Body{unitBlock(t)}}); err == nil {
		t.Error("zero angle should be rejected")
	}
}

// TestAssemblyRevolveNoAxisNoCenterlineFails: no explicit axis and no sketch centerline is an
// ambiguous input, reported as an error.
func TestAssemblyRevolveNoAxisNoCenterlineFails(t *testing.T) {
	f := NewAssemblyRevolveFeature(offsetSquareSketch(2, 2), 0, nil, ops.Cut, fullTurn)
	if _, err := f.Recompute(Input{Bodies: []*topo.Body{unitBlock(t)}}); err == nil {
		t.Error("revolve with no axis and no centerline should fail")
	}
}
