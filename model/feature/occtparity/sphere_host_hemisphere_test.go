// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The S6/S7 sphere-host hemisphere gate (sphere-notch-report.md). OCCT leaves the boss hemisphere
// UNTRIMMED on both cases — its blend patches terminate ON the footprint rim (the equator, a
// plane∩sphere circle): the shared contact edges' pcurves on the sphere are v=0 equator arcs, and
// the sphere face's own region is the full hemisphere (DRAWEXE 8.0.0 `sprops … 1.e-12` = 1061.86 =
// 2π·13², CoG z = R/2, on BOTH S6 and S7 — re-verified live, this slice). Our shipped topology has
// ALWAYS matched that: the sphere face's loop is the full equator (subdivided by the runout
// terminations at (±5,−12,0), the exact contact-line crossings OCCT also has) plus the parametric
// seam doubled — the "~0.55 notch" two review waves recorded was a MIS-READING: the loop's summed
// chord exceeds 2πR by exactly the seam counted twice (2×18.3848), and the missing area was
// spherePatchMesh's patchGridCap-clamped interior density, flat at every chord tolerance
// (a RESOLUTION defect, not a REGION defect). This gate pins both truths separately so neither
// regresses silently again:
//
//   - REGION: the loop is the full equator + one doubled seam (every non-seam boundary edge lies ON
//     the equator circle), and the PropertyQuality mesh covers the hemisphere EXACTLY once — its
//     summed signed solid angle at the centre is 2π (a notch, hole, fold or double-cover moves it).
//   - RESOLUTION: the mesh area meets PropertyQuality's own documented ~0.01% contract against the
//     closed form 2π·13² = 1061.858347 (ceiling 1.1× the measured post-fix deficit; the clamped
//     stereo-CDT path read −0.66, 6× outside contract, and fails this loud).
const (
	sphereHostAreaCeil   = 0.07 // 1.1× the measured fan deficit (S6 0.0632, S7 0.0622) — abs, model units²
	sphereHostOnEquator  = 1e-9 // abs |z| and |dist−R| bound for rim samples (measured ≤ 3.6e-15)
	sphereHostSolidAngle = 1e-6 // rel bound on |ΣΩ − 2π| (measured ≤ 2e-10)
)

func TestSphereHostHemisphereStaysFullAndMeshesToItsArea(t *testing.T) {
	dir := CorpusFixtureDir()
	for _, r := range Corpus() {
		if r.Grid != "simple" || (r.Case != "S6" && r.Case != "S7") {
			continue
		}
		body, ok := shippedCaseBody(r, dir)
		if !ok {
			t.Fatalf("%s/%s: no shipped body", r.Grid, r.Case)
		}
		sphereFace := findHostSphereFace(t, r.Case, body)
		assertSphereLoopIsFullEquator(t, r.Case, sphereFace)
		assertSphereMeshCoversAndMeasures(t, r.Case, sphereFace)
	}
}

// findHostSphereFace returns the case's single geom.Sphere face (the R=13 boss hemisphere).
func findHostSphereFace(t *testing.T, name string, body *topo.Body) *topo.Face {
	t.Helper()
	var found *topo.Face
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Sphere); ok {
			if found != nil {
				t.Fatalf("%s: more than one sphere face", name)
			}
			found = f
		}
	}
	if found == nil {
		t.Fatalf("%s: no sphere face on the shipped body", name)
	}
	return found
}

// assertSphereLoopIsFullEquator pins the REGION at the B-rep level: one loop, exactly one doubled
// (seam) edge, and every other edge's curve ON the equator circle (|z| and |dist−R| ≤ 1e-9 over 17
// samples). A runout that trimmed the sphere above its equator — the "notch" the old record claimed —
// puts boundary samples off the equator and fails here.
func assertSphereLoopIsFullEquator(t *testing.T, name string, f *topo.Face) {
	t.Helper()
	sp := f.Geometry().(geom.Sphere)
	loops := f.Loops()
	if len(loops) != 1 {
		t.Fatalf("%s: sphere face has %d loops, want 1 (a hole would be a real notch)", name, len(loops))
	}
	seamUses := 0
	for _, u := range loops[0].EdgeUses() {
		if edgeUseCount(loops[0], u.Edge()) == 2 {
			seamUses++
			continue // the parametric seam meridian — off the equator by construction
		}
		if off := equatorDeparture(u.Edge(), sp); off > sphereHostOnEquator {
			t.Errorf("%s: sphere boundary edge %d leaves the equator by %.3e (> %.0e) — the sphere is trimmed off its rim",
				name, u.Edge().ID(), off, sphereHostOnEquator)
		}
	}
	if seamUses != 2 {
		t.Errorf("%s: sphere loop has %d seam edge-uses, want exactly 2 (one doubled meridian)", name, seamUses)
	}
}

// edgeUseCount returns how many of the loop's uses reference e.
func edgeUseCount(l *topo.Loop, e *topo.Edge) int {
	n := 0
	for _, u := range l.EdgeUses() {
		if u.Edge() == e {
			n++
		}
	}
	return n
}

// equatorDeparture is the worst |z| or |dist(centre)−R| over 17 samples of the edge's curve — zero
// exactly when the curve lies on the equator circle of the (origin-centred, +z-axis) boss sphere.
func equatorDeparture(e *topo.Edge, sp geom.Sphere) float64 {
	c := e.Geometry()
	if c == nil {
		return stdmath.Inf(1)
	}
	lo, hi := c.Domain()
	worst := 0.0
	for i := 0; i <= 16; i++ {
		p := c.PointAt(lo + float64(i)/16*(hi-lo))
		off := stdmath.Max(stdmath.Abs(float64(p.Z)), stdmath.Abs(float64(sp.Center.DistanceTo(p))-sp.Radius))
		worst = stdmath.Max(worst, off)
	}
	return worst
}

// assertSphereMeshCoversAndMeasures pins the MESH: the PropertyQuality tessellation must cover the
// hemisphere exactly once (signed solid angle 2π) and read within sphereHostAreaCeil of the closed
// form — the resolution half of the defect (the clamped patch path read −0.66 here at EVERY swept
// chord tolerance; the latitude-ring fan reads −0.063).
func assertSphereMeshCoversAndMeasures(t *testing.T, name string, f *topo.Face) {
	t.Helper()
	sp := f.Geometry().(geom.Sphere)
	mesh := ops.TessellateFace(f, ops.PropertyQuality())
	closed := 2 * stdmath.Pi * sp.Radius * sp.Radius
	if got := ops.MeshArea(mesh); stdmath.Abs(got-closed) > sphereHostAreaCeil {
		t.Errorf("%s: sphere face meshes %.6f vs closed 2πR²=%.6f (|Δ|=%.4f > %.2f) — the density-capped path is back",
			name, got, closed, stdmath.Abs(got-closed), sphereHostAreaCeil)
	}
	omega := meshSolidAngleAt(mesh, sp.Center)
	if rel := stdmath.Abs(omega-2*stdmath.Pi) / (2 * stdmath.Pi); rel > sphereHostSolidAngle {
		t.Errorf("%s: mesh covers solid angle %.9f, want 2π=%.9f (rel %.3g) — a region defect (notch/fold/overlap)",
			name, omega, 2*stdmath.Pi, rel)
	}
}

// meshSolidAngleAt sums each triangle's signed solid angle at o (van Oosterom–Strackee): a single
// consistent cover of a hemisphere sums to ±2π; a notch falls short, an overlap or fold cancels.
func meshSolidAngleAt(mesh *ops.Mesh, o math.Point3) float64 {
	total := 0.0
	for k := 0; k+2 < len(mesh.Indices); k += 3 {
		total += triSolidAngle(o, mesh.Positions[mesh.Indices[k]], mesh.Positions[mesh.Indices[k+1]], mesh.Positions[mesh.Indices[k+2]])
	}
	return stdmath.Abs(total)
}

func triSolidAngle(o, a, b, c math.Point3) float64 {
	va, vb, vc := o.VectorTo(a), o.VectorTo(b), o.VectorTo(c)
	la, lb, lc := float64(va.Length()), float64(vb.Length()), float64(vc.Length())
	num := float64(va.Cross(vb).Dot(vc))
	den := la*lb*lc + float64(va.Dot(vb))*lc + float64(vb.Dot(vc))*la + float64(vc.Dot(va))*lb
	return 2 * stdmath.Atan2(num, den)
}
