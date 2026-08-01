// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A2 wave: the SPIRIC OPEN-ARC canal arm (fillet_spiric_rim_open.go), exercised on the real
// simple/E6 E8 F1 F3 fixtures — R4's torus-corner PATCH already certifies the sphere at each end
// (fillet_torus_corner_test.go); this file certifies the ARM's own station geometry, independent of
// whether the downstream corner weld (a separate, unbuilt capability — see wave-report-A2.md) accepts
// a canal/BSplineSurface arm yet.

// TestSpiricArcSweep_ResolvesTheRealArcDirection pins spiricArcSweep's core contract with hand-picked
// angles: ψ_mid strictly between ψ0 and ψ1 going CCW selects dir=+1 with the CCW span; swapping which
// side ψ_mid sits on selects dir=−1 with the CW span instead — proving the "which way around the tube"
// decision is driven by the actual arc content, not an arbitrary default.
func TestSpiricArcSweep_ResolvesTheRealArcDirection(t *testing.T) {
	const eighthPi = stdmath.Pi / 4
	// ψ0=0, ψ_mid=π/4 (CCW of ψ0), ψ1=π/2: the short CCW arc [0, π/2] contains π/4.
	if dir, span, ok := spiricArcSweep(0, eighthPi, stdmath.Pi/2); !ok || dir != 1 || stdmath.Abs(span-stdmath.Pi/2) > 1e-9 {
		t.Fatalf("CCW case: dir=%g span=%g ok=%v, want dir=1 span=π/2", dir, span, ok)
	}
	// Same ψ0/ψ1, but ψ_mid on the OTHER (CW, long-way-round) side: must flip to dir=-1, the CW span.
	midCW := stdmath.Pi + eighthPi // strictly between ψ1=π/2 and ψ0=0 going CW (through π, 3π/2...)
	if dir, span, ok := spiricArcSweep(0, midCW, stdmath.Pi/2); !ok || dir != -1 || stdmath.Abs(span-(2*stdmath.Pi-stdmath.Pi/2)) > 1e-9 {
		t.Fatalf("CW case: dir=%g span=%g ok=%v, want dir=-1 span=%g", dir, span, ok, 2*stdmath.Pi-stdmath.Pi/2)
	}
}

// TestSpiricArcSweep_RejectsMidOutsideEitherPath is the do-no-harm guard: a ψ_mid that lands on
// neither the CCW nor the CW arc between ψ0 and ψ1 (impossible for a genuine simple arc, but a
// defensive input) must decline, never silently pick a direction that does not contain it.
func TestSpiricArcSweep_RejectsMidOutsideEitherPath(t *testing.T) {
	// ψ0=ψ1 (a zero-length arc): neither direction has a positive span, so no direction can bracket
	// any interior point.
	if _, _, ok := spiricArcSweep(0, stdmath.Pi/4, 0); ok {
		t.Fatal("degenerate ψ0=ψ1 arc must decline, not silently accept a direction")
	}
}

// findOpenSpiricArmEdge scans a body for the OPEN meridian-cut Torus∧Plane arc — E6/E8/F1/F3's torus
// SECTOR rim, bounded between two corner vertices (not the closed J3/A4-style full loop).
func findOpenSpiricArmEdge(t *testing.T, body *topo.Body) *topo.Edge {
	t.Helper()
	res := ResolutionForBody(body)
	for _, e := range body.Edges() {
		if isClosedCircleEdge(e) {
			continue
		}
		host, pl, planeFace, ok := torusPlaneEdge(e)
		if !ok {
			continue
		}
		n, err := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
		if err != nil {
			continue
		}
		if spiricMeridianCut(host.AxisDir.Dot(n), torusRimRadius(e, host), res) {
			return e
		}
	}
	t.Fatal("no open meridian-cut Torus∧Plane arc found in the imported body")
	return nil
}

// TestSpiricOpenArcArmEdge_RealFixtures builds the open spiric arm on each of the four real R4-wave
// torus corpus fixtures (E6/E8 horn torus Rm=Rt=100, F1/F3 ring torus Rm=200 Rt=50) and independently
// certifies every station: the tube foot sits on the REAL imported host torus (checked via the
// torus's own implicit equation, not by calling back into spine.station), the plane foot sits on the
// REAL imported cap plane, and both feet sit at exactly radius r from the ball centre — the same
// independent-certificate rigor TestSpiricStationExactness applies to the closed J3 spine.
func TestSpiricOpenArcArmEdge_RealFixtures(t *testing.T) {
	for _, name := range []string{"E6", "E8", "F1", "F3"} {
		t.Run(name, func(t *testing.T) {
			body := corpusFixture(t, "simple/"+name+".step")
			e := findOpenSpiricArmEdge(t, body)
			host, pl, hostFace, planeFace, ok := torusPlaneEdgeFaces(e)
			if !ok {
				t.Fatalf("%s: arm edge is not a Torus∧Plane pair", name)
			}
			const r = 10.0
			spine, ok := newSpiricRimSpine(nil, e, host, pl, hostFace, planeFace, r, false)
			if !ok {
				t.Fatalf("%s: newSpiricRimSpine(open) declined", name)
			}
			ef, handled := spiricOpenArcArmEdge(body, e, filletPick{edge: e, r0: r, r1: r})
			if !handled {
				t.Fatalf("%s: spiricOpenArcArmEdge declined to build the arm", name)
			}
			surf, ok := ef.armSurface.(geom.BSplineSurface)
			if !ok {
				t.Fatalf("%s: arm surface is %T, want geom.BSplineSurface", name, ef.armSurface)
			}
			assertOpenSpiricArmExact(t, name, spine, host, pl, surf)
			assertArmMeshesFoldFree(t, name, surf)
		})
	}
}

// assertOpenSpiricArmExact samples the built surface's own v-columns (via the loft's chord
// parametrisation, mirroring assertLoftExactAtStations' approach) and checks, at u=0 and u=1, the
// INDEPENDENT torus/plane membership + radius-r identities against the REAL host geometry — not the
// spine's own station() formula, so a bug shared between the spine and this check would still be
// caught by disagreement with the real geom.Torus/geom.Plane objects.
func assertOpenSpiricArmExact(t *testing.T, name string, spine spiricRimSpine, host geom.Torus, pl geom.Plane, surf geom.BSplineSurface) {
	t.Helper()
	// The loft's BETWEEN-station error is bounded to spiricRimEnvelopeCoef·weld (the same envelope
	// resolveSpiricOpenArcStations refines to), not the raw weld — a sampled interior v may legitimately
	// sit anywhere inside that band even though every STATION column is exact to weld.
	weld := spiricRimEnvelopeCoef * ResolutionForSize(spine.host.MajorRadius+spine.host.MinorRadius).Weld()
	for i := 0; i <= 10; i++ {
		v := float64(i) / 10
		wallPt := surf.PointAt(0, v)
		planePt := surf.PointAt(1, v)
		// tube foot ON the real host torus: the implicit torus equation (√(x²+y²)−R)²+z² = a².
		u, vv := host.ParamAt(wallPt)
		if d := float64(host.PointAt(u, vv).DistanceTo(wallPt)); d > weld {
			t.Fatalf("%s v=%.2f: wall rail point %v is %.3g off the REAL host torus (want ≤ %.3g)", name, v, wallPt, d, weld)
		}
		// plane foot ON the real cap plane.
		if d := stdmath.Abs(float64(pl.Origin.VectorTo(planePt).Dot(pl.Normal())) / float64(pl.Normal().Length())); d > weld {
			t.Fatalf("%s v=%.2f: plane rail point %v is %.3g off the REAL cap plane (want ≤ %.3g)", name, v, planePt, d, weld)
		}
	}
}

// TestNewSpiricRimSpine_ClosedLoopGuardStillFires proves relaxing the guard for the open caller did
// NOT weaken it for the closed one: a spine whose offset plane exits the offset torus (R − b ≤ |w|,
// the exact guard assembleSpiricSpine checks) must still decline with requireClosedLoop=true — the
// mutation witness for the A2 refactor (a shared function split by a boolean, not two copies that
// could silently diverge).
func TestNewSpiricRimSpine_ClosedLoopGuardStillFires(t *testing.T) {
	// A horn torus (Rm=Rt=100): with r=10, convex (side=-1), b = a - r = 90, R - b = 100 - 90 = 10.
	// |w| = |capD + side*r| = |0 - 10| = 10, so R - b (=10) <= |w| (=10) -- exactly the guard's fail
	// condition (not one closed loop around the tube).
	host := weTestTorus(t, 100, 100)
	res := ResolutionForSize(200)
	n, err := math.UnitVector3FromVector(math.V3(1, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	e, hostFace := hornTorusMeridianEdge(t, host)
	if _, ok := assembleSpiricSpine(e, host, hostFace, n, 0, 10, res, true); ok {
		t.Fatal("requireClosedLoop=true must still reject a spine whose offset plane exits the offset torus")
	}
	if _, ok := assembleSpiricSpine(e, host, hostFace, n, 0, 10, res, false); !ok {
		t.Fatal("requireClosedLoop=false must accept the SAME horn-torus spine (only the visited stations need to exist)")
	}
}

// hornTorusMeridianEdge builds a minimal synthetic meridian-arc edge (a quarter-turn tube arc) ON a
// real torus FACE (so tubeMaterialSign can read its material-outward normal), purely to give
// assembleSpiricSpine a valid ClassifyEdgeConvexity/tubeMaterialSign/rimCircleCenter input — its own
// geometry is not what TestNewSpiricRimSpine_ClosedLoopGuardStillFires is testing.
func hornTorusMeridianEdge(t *testing.T, host geom.Torus) (*topo.Edge, *topo.Face) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "spiric-open-guard", 0))
	bld := topo.NewBuilder(true, lin)
	center := math.P3(100, 0, 0)
	circ, err := geom.NewCircle(center, math.V3(1, 0, 0), 100)
	if err != nil {
		t.Fatalf("rim circle: %v", err)
	}
	v0 := bld.AddVertex(math.P3(100, 0, 100), lin)
	v1 := bld.AddVertex(math.P3(100, 100, 0), lin)
	e := bld.AddEdge(circ, v0, v1, lin)
	hostFace := bld.AddFace(host, lin)
	return e, hostFace
}
