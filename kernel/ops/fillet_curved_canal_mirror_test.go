// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// F3a — the s_10 canal-arm surface MIRROR. The trihedral N7 corner is NON-CONCURRENT (canal family): its
// single blend ball C=(45,5.279,15) lies on the two CONVEX arms' spines only, never on the s_10 arm's own
// offset spine. cornerAt USED to override s_10's frame-derived near-corner centre with that blend ball,
// building the arm cylinder on the mirrored x=45 line instead of the oracle x=55 line (result_12). These
// tests drive the REAL production path (not the oracle-correct fixture surfaces the other canal tests feed)
// so the mirror is exercised end-to-end; the fix gates the override on spine concurrence.

// s10RealArm builds the three real N7 arm fillets through the production corner+fillet solve (the path that
// mirrored s_10) and returns the s_10 arm's rolling-ball cylinder — the ONLY arm whose surface is the plain
// Plane∧Plane cyl (armSurface stays nil until the weld; the two Plane∧Cylinder arms carry a torus/cylinder
// armSurface). It is the surface solvedEdgeFillet built from cornerAt's near-corner centre.
func s10RealArm(t *testing.T) geom.Cylinder {
	t.Helper()
	body := importCorpusSolid(t, "simple/N7")
	corner := vertexNear(t, body, math.P3(50, 0, 10))
	picks := make([]EdgeFilletRadii, 0, 3)
	for _, e := range corner.Edges() {
		picks = append(picks, EdgeFilletRadii{Key: e.ReferenceKey(), R0: 5, R1: 5})
	}
	edges, err := resolveFilletPicks(body, picks)
	if err != nil {
		t.Fatalf("resolveFilletPicks: %v", err)
	}
	blends, miters, err := computeCorners(body, edges)
	if err != nil {
		t.Fatalf("computeCorners: %v", err)
	}
	fils, err := computeFillets(body, edges, blends, miters, FillConcaveOutward, nil)
	if err != nil {
		t.Fatalf("computeFillets: %v", err)
	}
	for _, ef := range fils {
		if ef.armSurface == nil { // s_10: the Plane∧Plane arm, whose surface is its rolling-ball cyl
			return ef.cyl
		}
	}
	t.Fatal("no Plane∧Plane (s_10) arm found among the N7 corner fillets")
	return geom.Cylinder{}
}

// TestS10ArmSpineUnmirrored is the RED→GREEN mirror gate: the s_10 arm cylinder's spine must be the
// oracle x=55 line (result_12: origin (55,0,15), axis ŷ, R=5), NOT the mirrored x=45 line. It asserts the
// SPINE (any axis point has x=55, z=15, and the axis is ±ŷ) rather than the origin's y, since the origin's
// station along the axis is free.
func TestS10ArmSpineUnmirrored(t *testing.T) {
	cyl := s10RealArm(t)
	if got := cyl.Origin.X; stdmath.Abs(float64(got)-55) > 1e-6 {
		t.Fatalf("s_10 arm spine x = %.6f, want 55 (mirrored to 45 is the F3a defect)", got)
	}
	if got := cyl.Origin.Z; stdmath.Abs(float64(got)-15) > 1e-6 {
		t.Fatalf("s_10 arm spine z = %.6f, want 15", got)
	}
	axis := cyl.AxisDir.AsVector()
	if ay := stdmath.Abs(float64(axis.Y)); stdmath.Abs(ay-1) > 1e-9 {
		t.Fatalf("s_10 arm axis = %v, want ±ŷ", axis)
	}
	if stdmath.Abs(float64(cyl.Radius)-5) > 1e-9 {
		t.Fatalf("s_10 arm radius = %.6f, want 5", cyl.Radius)
	}
	// The z=10 host rail foot rides the un-mirrored spine to (55,30,10) on the loop — off-loop (45,30,10)
	// was the B=5.0 canal-host-bite floor. Confirm the spine passes through x=55 at the foot's z=10 plane.
	foot := math.P3(float64(cyl.Origin.X), 30, 10)
	if stdmath.Abs(float64(foot.X)-55) > 1e-6 {
		t.Fatalf("z=10 rail foot x = %.6f, want 55 (on the real z=10 loop)", foot.X)
	}
}

// TestN7CanalWeldsToValidSolid proves F3 assembles the REAL N7 fillet through the production entry point
// into a WATERTIGHT solid — the culmination of the un-mirror fix (F3a) + the whole-body assembly (F3). It
// must NOT floor (neither on the mirrored s_10 host bite "far span will not close", nor on the W3 "final
// weld not yet assembled" floor F3 removed) and the result must be a valid closed manifold solid. This is
// the kernel-level analogue of the corpus area gate TestOCCTBlendSimple/N7.
func TestN7CanalWeldsToValidSolid(t *testing.T) {
	body := importCorpusSolid(t, "simple/N7")
	corner := vertexNear(t, body, math.P3(50, 0, 10))
	keys := make([][]byte, 0, 3)
	for _, e := range corner.Edges() {
		keys = append(keys, e.ReferenceKey())
	}
	welded, err := FilletEdges(body, keys, 5)
	if err != nil {
		t.Fatalf("N7 canal weld must now succeed (F3), got floor: %v", err)
	}
	if rep := Validate(welded); !rep.Valid || !rep.Closed || !welded.IsSolid() {
		t.Fatalf("N7 canal weld is not a watertight solid: valid=%v closed=%v solid=%v issues=%v",
			rep.Valid, rep.Closed, welded.IsSolid(), rep.Issues)
	}
}

// TestResolveArmCentreDeclinesOnMirror hardens resolveArmCentre's former SILENT fallback: fed a MIRRORED
// arm surface (spine on the wrong side of the reflected candidate), it must DECLINE with the measured
// centres — not silently substitute w.center (which self-consistently lay on the mirrored spine and masked
// the s_10 defect). It reuses the real N7 fixture and mirrors the s_10 arm surface to x=45 (its axial spine
// on the wrong side): the reflected candidate C′=(55,·,15) is then off that mirrored spine, so no honest
// centre exists and the resolver must decline rather than fall back to w.center.
func TestResolveArmCentreDeclinesOnMirror(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	arms[1].armSurface = mustCylinder(t, math.P3(45, 0, 15), math.V3(0, 1, 0), 5) // MIRRORED s_10 (true x=55)
	scale := tangentCornerScale(w, arms)
	_, ok, reason := reflectedArmCentres(w, arms, scale, res)
	if ok {
		t.Fatal("mirrored s_10 surface must make reflectedArmCentres DECLINE (no silent w.center fallback)")
	}
	if reason == "" || !strings.Contains(reason, "off its own spine") {
		t.Fatalf("decline must carry the measured off-spine values, got %q", reason)
	}
}
