// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"os"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The CLOSED elliptic-rim canal band's gates, driven on the REAL corpus fixtures (the same STEP files
// occtparity scores) rather than synthetic geometry, because the defect this slice fixes lives in the
// IMPORTED oblique-extrusion representation — including the unreliable Reversed flag the convexity gate
// exists to route around. kernel/ops reads them directly (occtparity imports kernel/ops, so importing it
// back would cycle), the same way the U4 obstacle tests do.

const ellipticFixtureDir = "../../model/feature/occtparity/fixtures/simple/"

// importEllipticFixture imports one corpus fixture as a single body.
func importEllipticFixture(t *testing.T, name string) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(ellipticFixtureDir + name + ".step")
	if err != nil {
		t.Fatalf("read %s fixture: %v", name, err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import %s: %v (n=%d)", name, err, len(bodies))
	}
	return bodies[0]
}

// closedEllipticRimNear returns the fixture's CLOSED EllipticalCylinder∧Plane rim whose arc-length
// centroid is nearest `want` — how each test names the exact rim the corpus picks (a fixture carries
// two such rims, one per cap, and they differ only in position).
func closedEllipticRimNear(t *testing.T, b *topo.Body, want math.Point3) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestD := stdmath.Inf(1)
	for _, e := range b.Edges() {
		if e.StartVertex() != e.EndVertex() {
			continue
		}
		if _, _, _, _, ok := ellipticalCylinderPlaneEdge(e); !ok {
			continue
		}
		if d := float64(rimCentroid(e).DistanceTo(want)); d < bestD {
			bestD, best = d, e
		}
	}
	if best == nil || bestD > 1e-6*ellipticFixtureExtent(b) {
		t.Fatalf("no closed elliptic rim near %v (closest %.6g)", want, bestD)
	}
	return best
}

// rimCentroid is a closed edge's chord-sampled centroid.
func rimCentroid(e *topo.Edge) math.Point3 {
	c := e.Geometry()
	lo, hi := c.Domain()
	var acc math.Vector3
	const n = 64
	for i := 0; i < n; i++ {
		acc = acc.Add(math.P3(0, 0, 0).VectorTo(c.PointAt(lo + (hi-lo)*float64(i)/float64(n))))
	}
	return math.P3(0, 0, 0).TranslateBy(acc.Scale(1.0 / n))
}

// ellipticFixtureExtent is the body's vertex bounding-box diagonal — the scale the rim match is relative to.
func ellipticFixtureExtent(b *topo.Body) float64 {
	box := math.EmptyBox()
	for _, v := range b.Vertices() {
		box = box.ExtendPoint(v.Point())
	}
	return float64(box.Max.DistanceTo(box.Min))
}

// TestEllipticRimSpineStationsAreExactlyTangent is the construction's own certificate: at every station
// the ball centre must be at distance EXACTLY r from the cap plane and from the WALL SURFACE (the true
// point-inversion distance, not just from the algebraic foot), and both feet exactly r from the centre.
// This is what makes the lofted band an envelope rather than a plausible-looking tube.
func TestEllipticRimSpineStationsAreExactlyTangent(t *testing.T) {
	body := importEllipticFixture(t, "J6")
	rim := closedEllipticRimNear(t, body, math.P3(20, 0, 100))
	ec, pl, wallF, _, ok := ellipticalCylinderPlaneEdge(rim)
	if !ok {
		t.Fatal("J6 top rim is not an EllipticalCylinder∧Plane edge")
	}
	const r = 10
	spine, ok := newEllipticRimSpine(body, rim, ec, pl, wallF, r)
	if !ok {
		t.Fatal("newEllipticRimSpine declined J6's top rim")
	}
	tol := 1e-9 * r
	for k := 0; k < 64; k++ {
		u := 2 * stdmath.Pi * float64(k) / 64
		c, wf, pf, ok := spine.station(u)
		if !ok {
			t.Fatalf("station %d (u=%.4f) failed", k, u)
		}
		assertScalarNear(t, "wall foot", float64(wf.DistanceTo(c)), r, tol)
		assertScalarNear(t, "plane foot", float64(pf.DistanceTo(c)), r, tol)
		assertScalarNear(t, "plane distance", stdmath.Abs(float64(math.P3(0, 0, 0).VectorTo(c).Dot(pl.Normal()))-spine.cPl), r, tol)
		assertScalarNear(t, "wall distance", spine.tangencyError(c), 0, tol)
	}
}

// TestEllipticRimConvexityIsReadFromTheSolidNotTheReversedFlag pins the gate this slice adds. The
// imported oblique-extrusion elliptic face carries an unreliable Reversed flag, so ClassifyEdgeConvexity
// MIS-CALLS both of these rims (J6's convex cap rim reads "concave", T5's concave boss-base rim reads
// "convex"). The spine derives the side by probing the SOLID instead, and must get both right — J6 must
// come out CONVEX (side −1, ball inside the material) and T5 CONCAVE (side +1, ball in the void). Without
// this, a concave elliptic rim would silently build a wrong-sided convex band.
func TestEllipticRimConvexityIsReadFromTheSolidNotTheReversedFlag(t *testing.T) {
	for _, tc := range []struct {
		name       string
		rimAt      math.Point3
		radius     float64
		wantSide   float64
		classifier EdgeConvexity
	}{
		{"J6", math.P3(20, 0, 100), 10, -1, EdgeConcave},
		{"T5", math.P3(0, 0, 0), 4, +1, EdgeConvex},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := importEllipticFixture(t, tc.name)
			rim := closedEllipticRimNear(t, body, tc.rimAt)
			if got := ClassifyEdgeConvexity(rim); got != tc.classifier {
				t.Fatalf("premise changed: ClassifyEdgeConvexity(%s rim) = %v, want the MIS-CALL %v "+
					"(this test exists because the imported elliptic face's Reversed flag is unreliable)", tc.name, got, tc.classifier)
			}
			ec, pl, wallF, _, _ := ellipticalCylinderPlaneEdge(rim)
			spine, ok := newEllipticRimSpine(body, rim, ec, pl, wallF, tc.radius)
			if !ok {
				t.Fatalf("newEllipticRimSpine declined the %s rim", tc.name)
			}
			if spine.side != tc.wantSide {
				t.Errorf("%s rim side = %v, want %v (the solid probe must override the Reversed-flag classifier)",
					tc.name, spine.side, tc.wantSide)
			}
		})
	}
}

// TestEllipticClosedRimDeclinesSpillingBand is the do-no-harm floor for T5/U2: their fillet foot ring
// runs OFF the plate it stands on (OCCT answers with a clipped, multi-piece band — a different
// construction), so the arm must return handled=false and let the edge fall through to the unchanged
// flat refusal rather than ship a band poking through the plate's side walls.
func TestEllipticClosedRimDeclinesSpillingBand(t *testing.T) {
	for _, tc := range []struct {
		name   string
		rimAt  math.Point3
		radius float64
	}{
		{"T5", math.P3(0, 0, 0), 4},
		{"U2", math.P3(0, 0, 0), 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := importEllipticFixture(t, tc.name)
			rim := closedEllipticRimNear(t, body, tc.rimAt)
			if _, handled := ellipticClosedRimArmEdge(body, rim, filletPick{edge: rim, r0: tc.radius, r1: tc.radius}); handled {
				t.Errorf("%s: the spilling band was BUILT (want handled=false — the foot ring leaves the plate face)", tc.name)
			}
		})
	}
}

// TestEllipticClosedRimLeavesTheRulingEdgeAlone keeps F4 on its analytic path: an OPEN straight-ruling
// elliptic edge is ellipticalCylinderArmEdge's (an exact right circular cylinder), and the closed-rim
// canal must never steal it.
func TestEllipticClosedRimLeavesTheRulingEdgeAlone(t *testing.T) {
	body := importEllipticFixture(t, "F4")
	var ruling *topo.Edge
	for _, e := range body.Edges() {
		if e.StartVertex() == e.EndVertex() {
			continue
		}
		if _, _, _, _, ok := ellipticalCylinderPlaneEdge(e); ok {
			ruling = e
		}
	}
	if ruling == nil {
		t.Fatal("F4 has no open EllipticalCylinder∧Plane edge")
	}
	if _, handled := ellipticClosedRimArmEdge(body, ruling, filletPick{edge: ruling, r0: 10, r1: 10}); handled {
		t.Error("the closed-rim canal handled F4's OPEN ruling edge (want handled=false — that edge is the analytic cylinder arm's)")
	}
}

// TestEllipticRimCanalBandMatchesTheExactEnvelopeArea checks the built band against the area of the TRUE
// rolling-ball envelope, integrated independently (see elliptic-rims-report.md): 7240.851 for J6 and
// 4362.971 for J8. It is the numeric proof that the loft IS the envelope and not merely a smooth tube —
// and it is independent of OCCT, whose recorded whole-body number for these two cases is inflated by its
// own sprops mis-integration of the extrusion wall.
func TestEllipticRimCanalBandMatchesTheExactEnvelopeArea(t *testing.T) {
	for _, tc := range []struct {
		name      string
		rimAt     math.Point3
		radius    float64
		exactArea float64
	}{
		{"J6", math.P3(20, 0, 100), 10, 7240.851},
		{"J8", math.P3(0, 0, 0), 10, 4362.971},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := importEllipticFixture(t, tc.name)
			rim := closedEllipticRimNear(t, body, tc.rimAt)
			ec, pl, wallF, capF, _ := ellipticalCylinderPlaneEdge(rim)
			canal, ok := buildEllipticRimCanal(body, rim, ec, pl, wallF, capF, tc.radius)
			if !ok {
				t.Fatalf("%s: the canal band was not built", tc.name)
			}
			got := parametricSurfaceArea(canal.surf)
			if rel := stdmath.Abs(got-tc.exactArea) / tc.exactArea; rel > 1e-4 {
				t.Errorf("%s band area = %.6g, want the exact envelope %.6g (rel %.3g > 1e-4)", tc.name, got, tc.exactArea, rel)
			}
		})
	}
}

// parametricSurfaceArea integrates |∂P/∂u × ∂P/∂v| over the surface's full parameter box — the band's
// geometric area, read from the surface itself rather than from a tessellation.
func parametricSurfaceArea(s geom.BSplineSurface) float64 {
	const n = 240
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	du, dv := (uHi-uLo)/n, (vHi-vLo)/n
	total := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			pu, pv := s.DerivativesAt(uLo+(float64(i)+0.5)*du, vLo+(float64(j)+0.5)*dv)
			total += float64(pu.Cross(pv).Length()) * du * dv
		}
	}
	return total
}

// TestCanalBandMeshRecognisesOnlyTheVClosedBand pins the tessellation gate. The canal band closes along
// v (its last station repeats its first) and its two closed boundary edges are the u-boundary
// isocurves; an imported tube patch closes the OTHER way and must be left to the general NURBS meshers
// (lofting it rail-to-rail would crack it — TestImportedNurbsDuctWatertight).
func TestCanalBandMeshRecognisesOnlyTheVClosedBand(t *testing.T) {
	body := importEllipticFixture(t, "J6")
	rim := closedEllipticRimNear(t, body, math.P3(20, 0, 100))
	ec, pl, wallF, capF, _ := ellipticalCylinderPlaneEdge(rim)
	canal, ok := buildEllipticRimCanal(body, rim, ec, pl, wallF, capF, 10)
	if !ok {
		t.Fatal("the J6 canal band was not built")
	}
	if !closesAlongV(canal.surf) {
		t.Error("closesAlongV rejected the canal band (it repeats its first station last, so it MUST close along v)")
	}
	if closesAlongV(transposedSurface(canal.surf)) {
		t.Error("closesAlongV accepted the TRANSPOSED band — a u-closed tube patch must be left to the NURBS meshers")
	}
}

// transposedSurface swaps a surface's u and v roles, turning the v-closed canal band into the u-closed
// shape an imported tube patch presents — the negative case the tessellation gate must reject.
func transposedSurface(s geom.BSplineSurface) geom.BSplineSurface {
	ctrl := make([][]math.Point3, len(s.Ctrl[0]))
	weights := make([][]float64, len(s.Ctrl[0]))
	for j := range ctrl {
		ctrl[j] = make([]math.Point3, len(s.Ctrl))
		weights[j] = make([]float64, len(s.Ctrl))
		for i := range s.Ctrl {
			ctrl[j][i], weights[j][i] = s.Ctrl[i][j], s.Weights[i][j]
		}
	}
	return geom.BSplineSurface{
		UDegree: s.VDegree, VDegree: s.UDegree, Ctrl: ctrl, Weights: weights,
		UKnots: s.VKnots, VKnots: s.UKnots,
	}
}
