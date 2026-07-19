// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	stepio "oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CN2 — the Cone∧Plane RULING-edge canal arm (the crux). These tests pin the EXACT construction
// (cone-host-corner-derivation.md §2 "Arm B"): closed-form hyperbola stations, exact plane/cone feet
// (|foot − m| = r to 1e-9), the band-arc-only fold guard (D1's vertex would spuriously fold a full-tube
// check), and the loft's at-station exactness on REAL imported C2/C6/D1 ruling geometry — plus the arm
// meshes fold-free. CN2 greens no corpus case (the corner still declines "corner face must be planar").

const coneCanalExactTol = 1e-9

// coneCanalCase is a synthetic reproduction of a corpus ruling edge: the host cone (apex, axis −ẑ, tanα)
// and its radial plane's material-outward normal, from the DRAWEXE numbers.
type coneCanalCase struct {
	name     string
	apex     math.Point3
	tanAlpha float64
	nOut     math.Vector3 // radial-plane material-outward normal
	xfLo     float64      // a valid x_f sampling window (both ends admit the ball)
	xfHi     float64
}

// coneCanalCases are the three rim-ruling fixtures (C2/C6/D1); C8's two rulings share C2/D1's cone/plane
// math. The x_f windows sit inside each ruling's fittable span.
func coneCanalCases() []coneCanalCase {
	return []coneCanalCase{
		{"C2", math.P3(0, 0, 270), 1.0 / 3.0, math.V3(0, 1, 0), 20, 78},
		{"C6", math.P3(0, 0, 270), 1.0 / 3.0, math.V3(1, 0, 0), 20, 42},
		{"D1", math.P3(0, 0, 120), 5.0 / 12.0, math.V3(0, 1, 0), 0, 39},
	}
}

// coneCanalSpineFor builds the exact spine of a synthetic case.
func coneCanalSpineFor(t *testing.T, c coneCanalCase) coneCanalSpine {
	t.Helper()
	co, err := geom.NewCone(c.apex, coneAxisDown(), stdmath.Atan(c.tanAlpha))
	if err != nil {
		t.Fatalf("%s cone: %v", c.name, err)
	}
	nOut, err := math.UnitVector3FromVector(c.nOut)
	if err != nil {
		t.Fatalf("%s nOut: %v", c.name, err)
	}
	spine, reason := newConeCanalSpine(co, nOut, coneArmR, ResolutionForSize(300))
	if reason != coneArmBuilt {
		t.Fatalf("%s newConeCanalSpine declined a valid spine (reason %d)", c.name, reason)
	}
	return spine
}

// TestConeCanalSpine_ExactFeet pins the closed-form station geometry: the ball centre lies on the exact
// hyperbola and the r-offset plane, and BOTH host feet sit at distance r from it — the plane foot on the
// radial plane, the cone foot on the cone — all to ≤1e-9 (algebraically exact, not marched). |T − m| = r
// is the load-bearing identity ζ·sinα − ρ·cosα = r the canal is built on.
func TestConeCanalSpine_ExactFeet(t *testing.T) {
	for _, c := range coneCanalCases() {
		t.Run(c.name, func(t *testing.T) {
			s := coneCanalSpineFor(t, c)
			co, _ := geom.NewCone(c.apex, coneAxisDown(), stdmath.Atan(c.tanAlpha))
			for i := 0; i <= 12; i++ {
				xf := c.xfLo + (c.xfHi-c.xfLo)*float64(i)/12
				assertStationExact(t, c.name, s, co, xf)
			}
		})
	}
}

// assertStationExact checks one station's hyperbola membership, plane-foot, and cone-foot exactness.
func assertStationExact(t *testing.T, name string, s coneCanalSpine, co geom.Cone, xf float64) {
	t.Helper()
	m := s.center(xf)
	assertOnOffsetPlane(t, name, s, m)
	assertOnHyperbola(t, name, s, xf)
	fP := s.planeFoot(m)
	assertAtRadius(t, name+" planeFoot", m, fP)
	assertPointOnRadialPlane(t, name, s, fP)
	coneT, ok := s.coneFoot(m)
	if !ok {
		t.Fatalf("%s coneFoot declined at x_f=%g", name, xf)
	}
	assertAtRadius(t, name+" coneFoot", m, coneT)
	assertPointOnCone(t, name, co, coneT)
}

// assertOnOffsetPlane checks the ball centre sits at signed distance −r from the radial plane.
func assertOnOffsetPlane(t *testing.T, name string, s coneCanalSpine, m math.Point3) {
	t.Helper()
	if d := float64(s.apex.VectorTo(m).Dot(s.nOut)) + s.radius; stdmath.Abs(d) > coneCanalExactTol {
		t.Fatalf("%s centre off the r-offset plane by %g (want (m−A)·n̂ = −r)", name, d)
	}
}

// assertOnHyperbola checks the spine's defining relation x_f² = tan²α·(ζ − r/sinα)² − r².
func assertOnHyperbola(t *testing.T, name string, s coneCanalSpine, xf float64) {
	t.Helper()
	delta := s.zetaAt(xf) - s.radius/s.sinA
	want := s.tanA*s.tanA*delta*delta - s.radius*s.radius
	if d := stdmath.Abs(xf*xf - want); d > coneCanalExactTol {
		t.Fatalf("%s x_f=%g off the hyperbola by %g", name, xf, d)
	}
}

// assertAtRadius checks a foot sits exactly the ball radius from the centre.
func assertAtRadius(t *testing.T, what string, m, foot math.Point3) {
	t.Helper()
	if d := stdmath.Abs(float64(foot.DistanceTo(m)) - coneArmR); d > coneCanalExactTol {
		t.Fatalf("%s is %g off radius %g (|foot−m| must equal r)", what, d, coneArmR)
	}
}

// assertPointOnRadialPlane checks the plane foot lies on the radial plane through A with normal n̂.
func assertPointOnRadialPlane(t *testing.T, name string, s coneCanalSpine, p math.Point3) {
	t.Helper()
	if d := float64(s.apex.VectorTo(p).Dot(s.nOut)); stdmath.Abs(d) > coneCanalExactTol {
		t.Fatalf("%s plane foot off the radial plane by %g", name, d)
	}
}

// assertPointOnCone checks the cone foot lies on the host cone (signed distance
// (w·â)·sinα − |w⊥|·cosα == 0, w = point − apex).
func assertPointOnCone(t *testing.T, name string, co geom.Cone, p math.Point3) {
	t.Helper()
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	a := co.AxisDir.AsVector()
	w := co.Apex.VectorTo(p)
	axial := float64(w.Dot(a))
	perp := float64(w.Sub(a.Scale(axial)).Length())
	if d := axial*sinA - perp*cosA; stdmath.Abs(d) > coneCanalExactTol {
		t.Fatalf("%s cone foot off the host cone by %g", name, d)
	}
}

// TestConeCanalBandFoldGuard is the fold-guard SCOPING proof on D1 (tanα=5/12 ⇒ vertex curvature
// κ·r = cotα = 2.4 > 1): the FULL tube self-intersects at the hyperbola vertex (1 − κ·r ≤ 0 toward the
// concave apex side), but the fillet BAND arc (plane foot → up-tilted cone foot) never enters that lobe
// and stays regular (> 0). A full-tube check would spuriously reject D1 — this pins that the band-only
// guard accepts it while the full-tube factor is genuinely negative.
func TestConeCanalBandFoldGuard(t *testing.T) {
	s := coneCanalSpineFor(t, coneCanalCases()[2]) // D1
	m := s.center(0)                               // the vertex station
	fP := s.planeFoot(m)
	coneT, ok := s.coneFoot(m)
	if !ok {
		t.Fatal("D1 vertex coneFoot declined")
	}
	if got := s.bandArcMinRegularity(0, m, fP, coneT); got <= 0 {
		t.Fatalf("D1 vertex BAND arc regularity %g ≤ 0 — the band folded (should be positive ≈ 1+cosα)", got)
	}
	k := s.curvatureVector(0)
	nHat, _ := math.UnitVector3FromVector(k) // the concave (full-tube fold) direction
	fullTube := 1 - s.radius*float64(k.Dot(nHat.AsVector()))
	if fullTube >= 0 {
		t.Fatalf("mutation witness broken: D1 full-tube factor %g ≥ 0 — expected < 0 (κ·r=cotα=2.4>1)", fullTube)
	}
}

// TestConeCanalStationOf pins the closed-form armStation canal case: a point placed ON the spine at a
// known x_f is recovered exactly, and a point pushed off the r-offset plane is rejected (do-no-harm).
func TestConeCanalStationOf(t *testing.T) {
	s := coneCanalSpineFor(t, coneCanalCases()[0]) // C2
	res := ResolutionForSize(300)
	const xf = 42.0
	on := s.center(xf)
	got, ok := s.stationOf(on, res.Size(), res.Weld())
	if !ok || stdmath.Abs(got-xf) > coneCanalExactTol {
		t.Fatalf("stationOf on-spine: got (%g, %v), want (%g, true)", got, ok, xf)
	}
	off := on.TranslateBy(s.nOut.Scale(1.0)) // 1 unit off the r-offset plane
	if _, ok := s.stationOf(off, res.Size(), res.Weld()); ok {
		t.Fatal("stationOf accepted a centre 1 unit off the r-offset plane (want reject)")
	}
}

// TestConeCanalArm_RealImport builds the exact canal arm from the REAL imported C2/C6/D1 ruling edges and
// pins two properties: (1) the loft is EXACT at every station — surf.PointAt(u, v_j) sits at radius r
// from the exact hyperbola centre c_j to ≤ weld (global interpolation passes through the exact columns);
// (2) the trimmed BSpline arm meshes FOLD-FREE with positive area. This is the crux gate: a real
// non-analytic arm through the fillet engine, exact and fold-free.
func TestConeCanalArm_RealImport(t *testing.T) {
	for _, name := range []string{"C2", "C6", "D1"} {
		t.Run(name, func(t *testing.T) {
			body := importSimpleFixture(t, name)
			e := findConeRulingEdge(t, body)
			ef, handled, err := coneArmEdge(body, e, filletPick{edge: e, r0: coneArmR, r1: coneArmR})
			if !handled || err != nil || ef.armCanalSpine == nil {
				t.Fatalf("%s ruling edge: want built canal arm, got handled=%v err=%v spine=%v", name, handled, err, ef.armCanalSpine)
			}
			surf, ok := ef.armSurface.(geom.BSplineSurface)
			if !ok {
				t.Fatalf("%s canal arm is %T, want geom.BSplineSurface", name, ef.armSurface)
			}
			assertLoftExactAtStations(t, name, *ef.armCanalSpine, e, surf)
			assertArmMeshesFoldFree(t, name, surf)
		})
	}
}

// assertLoftExactAtStations recomputes the arm's exact stations and asserts the built surface passes
// through each station's radius-r arc: surf.PointAt(u, v_j) is at distance r from centre c_j to ≤ weld.
func assertLoftExactAtStations(t *testing.T, name string, s coneCanalSpine, e *topo.Edge, surf geom.BSplineSurface) {
	t.Helper()
	lo, hi, reason := s.edgeXfSpan(e, ResolutionForSize(300))
	if reason != coneArmBuilt {
		t.Fatalf("%s edgeXfSpan declined (reason %d)", name, reason)
	}
	centers, _, _, reason := s.sampleStations(lo, hi)
	if reason != coneArmBuilt {
		t.Fatalf("%s sampleStations declined (reason %d)", name, reason)
	}
	weld := ResolutionForSize(300).Weld()
	vparams := chordLengthParams(centers)
	for j, c := range centers {
		for _, u := range []float64{0, 0.3, 0.5, 0.7, 1} {
			if d := stdmath.Abs(float64(surf.PointAt(u, vparams[j]).DistanceTo(c)) - coneArmR); d > weld {
				t.Fatalf("%s station %d u=%g: surface point %g off radius r from the exact centre (weld %g)", name, j, u, d, weld)
			}
		}
	}
}

// chordLengthParams reproduces alphaParams(·, 1): the normalized cumulative chord length of the centres,
// which is the loft's v-parametrization — so v_j is where station j's exact column lives.
func chordLengthParams(pts []math.Point3) []float64 {
	cum := make([]float64, len(pts))
	for k := 1; k < len(pts); k++ {
		cum[k] = cum[k-1] + float64(pts[k-1].DistanceTo(pts[k]))
	}
	out := make([]float64, len(pts))
	for k := range out {
		out[k] = cum[k] / cum[len(pts)-1]
	}
	out[len(out)-1] = 1
	return out
}

// assertArmMeshesFoldFree tessellates the whole [0,1]² canal-arm patch through the production fold-driven
// trim path and asserts zero fold edges and a positive mesh area (the CLAUDE.md tessellation gate).
func assertArmMeshesFoldFree(t *testing.T, name string, surf geom.BSplineSurface) {
	t.Helper()
	u0, u1 := surf.UDomain()
	v0, v1 := surf.VDomain()
	outerUV := []math.Point2{math.P2(math.Scalar(u0), math.Scalar(v0)), math.P2(math.Scalar(u1), math.Scalar(v0)),
		math.P2(math.Scalar(u1), math.Scalar(v1)), math.P2(math.Scalar(u0), math.Scalar(v1))}
	outer3D := make([]math.Point3, len(outerUV))
	for i, p := range outerUV {
		outer3D[i] = surf.PointAt(float64(p.X), float64(p.Y))
	}
	su, sv := metricScale(surf)
	m := foldDrivenPatch(surf, su, sv, DefaultQuality(), outer3D, outerUV, nil, nil)
	if m == nil || m.TriangleCount() == 0 {
		t.Fatalf("%s canal arm produced no mesh", name)
	}
	if n := FoldEdgeCount(m); n != 0 {
		t.Fatalf("%s canal arm meshed with %d fold edges; want 0", name, n)
	}
	if a := MeshArea(m); a <= 0 || stdmath.IsInf(a, 0) || stdmath.IsNaN(a) {
		t.Fatalf("%s canal arm meshed to area %g; want finite positive", name, a)
	}
}

// importSimpleFixture loads a simple-grid OCCT input solid from the parity fixtures (relative to
// kernel/ops); the ruling arm is validated on the REAL imported cone/plane geometry, not a synthetic one.
func importSimpleFixture(t *testing.T, name string) *topo.Body {
	t.Helper()
	path := filepath.Join("..", "..", "model", "feature", "occtparity", "fixtures", "simple", name+".step")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	bodies, _, err := stepio.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import %s: %d bodies, err %v", path, len(bodies), err)
	}
	return bodies[0]
}

// findConeRulingEdge returns the first Cone∧Plane RULING edge (plane containing the axis) in the body.
func findConeRulingEdge(t *testing.T, body *topo.Body) *topo.Edge {
	t.Helper()
	res := ResolutionForBody(body)
	for _, e := range body.Edges() {
		co, pl, _, _, ok := conePlaneEdge(e)
		if ok && classifyConeArm(co, pl, coneRadiusAt(co, edgeMidpoint(e)), res) == coneClassRuling {
			return e
		}
	}
	t.Fatal("no Cone∧Plane ruling edge found in the imported body")
	return nil
}
