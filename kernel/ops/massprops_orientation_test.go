// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// Regression for Oblikovati/Oblikovati#1318: mass properties used to orient triangles from summed
// per-vertex shading normals, which flip at random on coarse high-curvature meshes (saddles,
// silhouette slivers) and corrupt the divergence-theorem sum. The orientation is now topological
// (consistentOutwardFlips), so volume/centroid/inertia converge cleanly with no sign-flip spikes.

// sphereVolume returns the analytic volume of a sphere of the given radius.
func sphereVolume(r float64) float64 { return 4.0 / 3.0 * math.Pi * r * r * r }

// TestCoarseSphereVolumeConvergesMonotonically tessellates a sphere at successively finer qualities
// and asserts the volume rises monotonically toward the analytic value from below (an inscribed
// polyhedron under-fills), with no spike. A random saddle flip would break monotonicity or sign.
//
// It measures meshGeometryProperties DIRECTLY, because that is its subject. It used to drive
// BodyGeometryProperties, which since M48/C3 (#3453) integrates the analytic B-rep and consults its
// Quality argument only for a fallback this body never takes — so all four qualities returned the
// SAME exact volume, the "error" was a constant at machine epsilon, and the convergence assertion
// compared 2.51e-16 against 5.03e-17 and tipped on floating-point noise (green on amd64, red on CI's
// arm64, where FMA contraction rounds differently). The mesh integrator is still the thing that can
// suffer an orientation spike, and it is still reached — as the fallback, and by every consumer of a
// mesh with no B-rep behind it.
func TestCoarseSphereVolumeConvergesMonotonically(t *testing.T) {
	t.Parallel()
	const r = 3.0
	want := sphereVolume(r)
	sphere, err := brep.SolidSphere(m.P3(0, 0, 0), r, "sphere")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	qualities := []Quality{
		{ChordTolerance: 0.5, AngleTolerance: 40 * math.Pi / 180},  // coarse
		{ChordTolerance: 0.1, AngleTolerance: 20 * math.Pi / 180},  // medium
		{ChordTolerance: 0.02, AngleTolerance: 8 * math.Pi / 180},  // fine
		{ChordTolerance: 0.005, AngleTolerance: 3 * math.Pi / 180}, // finer
	}
	var coarseErr, prevErr = 0.0, math.Inf(1)
	for i, q := range qualities {
		mesh, _ := TessellateBody(sphere, q)
		v := meshGeometryProperties(mesh).Volume
		if v <= 0 {
			t.Fatalf("q[%d]: non-positive volume %g (sign flip)", i, v)
		}
		if v > want*(1+1e-9) {
			t.Errorf("q[%d]: volume %g exceeds analytic %g (inscribed mesh must under-fill)", i, v, want)
		}
		relErr := (want - v) / want
		if relErr > prevErr+1e-12 {
			t.Errorf("q[%d]: rel error %g grew vs previous %g (non-monotone — orientation spike)", i, relErr, prevErr)
		}
		if i == 0 {
			coarseErr = relErr
		}
		prevErr = relErr
	}
	// Genuine convergence: refining the mesh ~100× (chord 0.5→0.005) must shrink the error by at
	// least 5×. A constant orientation-injected bias would not converge like this.
	if prevErr > coarseErr/5 {
		t.Errorf("finest rel error %g did not converge below coarse/5 = %g", prevErr, coarseErr/5)
	}
	if prevErr > 1e-2 {
		t.Errorf("finest rel error %g too large for a converged sphere mesh", prevErr)
	}
	assertAnalyticVolumeIgnoresQuality(t, sphere, want, qualities)
}

// assertAnalyticVolumeIgnoresQuality pins what replaced the convergence above: BodyGeometryProperties
// integrates the analytic B-rep, so a sphere's volume is EXACT at every quality rather than
// converging toward exactness. This is the stronger contract, and stating it here keeps the two
// meters distinguishable — the reason the convergence assertion had to move off this entry point.
func assertAnalyticVolumeIgnoresQuality(t *testing.T, sphere *topo.Body, want float64, qualities []Quality) {
	t.Helper()
	for i, q := range qualities {
		got := BodyGeometryProperties(sphere, q).Volume
		if rel := math.Abs(got-want) / want; rel > 1e-12 {
			t.Errorf("q[%d]: analytic volume %.17g vs exact %.17g (rel %.3g) — the analytic path must not "+
				"depend on tessellation quality", i, got, want, rel)
		}
	}
}

// TestCoarseSphereVolumeWithinChordBound asserts that even at the COARSEST quality the volume error
// is bounded by O(chord²)/R² — the geometric chord-deviation bound — not an O(1) spike. Pre-fix, a
// flipped saddle facet on a coarse sphere produced errors far beyond this bound.
func TestCoarseSphereVolumeWithinChordBound(t *testing.T) {
	t.Parallel()
	const r = 3.0
	want := sphereVolume(r)
	sphere, err := brep.SolidSphere(m.P3(0, 0, 0), r, "sphere")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	const chord = 0.5
	v := BodyGeometryProperties(sphere, Quality{ChordTolerance: chord, AngleTolerance: 40 * math.Pi / 180}).Volume
	// Shell bound: an inscribed polyhedron's volume deficit is at most the surface area times the
	// chord deviation (a shell of that thickness). A flipped saddle facet blows past this bound.
	area := 4 * math.Pi * r * r
	bound := area * chord
	if math.Abs(v-want) > bound {
		t.Errorf("coarse sphere volume %g, analytic %g, |err| %g exceeds shell bound area*chord = %g", v, want, math.Abs(v-want), bound)
	}
}

// TestSaddleRichTorusVolumeStable uses a torus — the canonical surface carrying both positive and
// NEGATIVE Gaussian curvature (its inner half is a saddle band) — as the saddle-rich oracle. Volume
// must track the analytic 2π²·R·r² across qualities with no spike. The summed-normal test was most
// fragile exactly on this saddle band.
func TestSaddleRichTorusVolumeStable(t *testing.T) {
	t.Parallel()
	const major, minor = 5.0, 1.5
	want := 2 * math.Pi * math.Pi * major * minor * minor
	torus, err := brep.SolidTorus(m.P3(0, 0, 0), m.V3(0, 0, 1), major, minor, "torus")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	qualities := []Quality{
		{ChordTolerance: 0.2, AngleTolerance: 20 * math.Pi / 180},
		{ChordTolerance: 0.05, AngleTolerance: 8 * math.Pi / 180},
		{ChordTolerance: 0.01, AngleTolerance: 3 * math.Pi / 180},
	}
	var prevErr = math.Inf(1)
	for i, q := range qualities {
		v := BodyGeometryProperties(torus, q).Volume
		if v <= 0 {
			t.Fatalf("q[%d]: non-positive torus volume %g (saddle sign flip)", i, v)
		}
		relErr := math.Abs(v-want) / want
		if relErr > prevErr+1e-3 {
			t.Errorf("q[%d]: torus rel error %g jumped above previous %g (saddle spike)", i, relErr, prevErr)
		}
		prevErr = relErr
	}
	if prevErr > 5e-3 {
		t.Errorf("finest torus rel error %g too large", prevErr)
	}
}

// TestSolidSphereInertiaIsotropic checks the inertia tensor of a uniform solid sphere: Ixx=Iyy=Izz=
// (2/5)·V·R² and the products of inertia vanish. A saddle-flipped covariance contribution would
// break the isotropy or the diagonal value.
func TestSolidSphereInertiaIsotropic(t *testing.T) {
	t.Parallel()
	const r = 3.0
	sphere, err := brep.SolidSphere(m.P3(0, 0, 0), r, "sphere")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	q := Quality{ChordTolerance: 0.01, AngleTolerance: 3 * math.Pi / 180}
	it := BodyInertia(sphere, q)
	v := BodyGeometryProperties(sphere, q).Volume
	want := 0.4 * v * r * r
	tol := want * 5e-3
	for _, c := range []struct {
		name string
		got  float64
	}{{"Ixx", it.Ixx}, {"Iyy", it.Iyy}, {"Izz", it.Izz}} {
		if math.Abs(c.got-want) > tol {
			t.Errorf("%s = %g, want %g (tol %g)", c.name, c.got, want, tol)
		}
	}
	offTol := want * 5e-3
	for _, c := range []struct {
		name string
		got  float64
	}{{"Ixy", it.Ixy}, {"Iyz", it.Iyz}, {"Izx", it.Izx}} {
		if math.Abs(c.got) > offTol {
			t.Errorf("%s = %g, want ~0 (tol %g)", c.name, c.got, offTol)
		}
	}
}

// inconsistentCubeMesh builds a unit-cube surface mesh of side s at the origin, then deliberately
// reverses the winding of a subset of its triangles and zeroes the per-vertex normals — a worst-case
// input for the OLD shading-normal orientation (which would then mis-sum the volume). The topological
// orientation must still recover |V| = s³.
func inconsistentCubeMesh(s float64) *Mesh {
	mesh := &Mesh{}
	c := [8]m.Point3{
		m.P3(0, 0, 0), m.P3(s, 0, 0), m.P3(s, s, 0), m.P3(0, s, 0),
		m.P3(0, 0, s), m.P3(s, 0, s), m.P3(s, s, s), m.P3(0, s, s),
	}
	for _, p := range c {
		mesh.addVertex(p, m.Vector3{}) // zeroed normals: prove they are unused
	}
	// Outward-wound quads (CCW seen from outside), each split into two triangles.
	quads := [6][4]int{
		{0, 3, 2, 1}, // bottom (z=0), normal -Z
		{4, 5, 6, 7}, // top (z=s),    normal +Z
		{0, 1, 5, 4}, // front (y=0),  normal -Y
		{2, 3, 7, 6}, // back (y=s),   normal +Y
		{1, 2, 6, 5}, // right (x=s),  normal +X
		{0, 4, 7, 3}, // left (x=0),   normal -X
	}
	for qi, q := range quads {
		t0 := [3]int{q[0], q[1], q[2]}
		t1 := [3]int{q[0], q[2], q[3]}
		if qi%2 == 0 { // corrupt every other quad's winding to make the input inconsistent
			t0[1], t0[2] = t0[2], t0[1]
			t1[1], t1[2] = t1[2], t1[1]
		}
		mesh.addTriangle(t0[0], t0[1], t0[2])
		mesh.addTriangle(t1[0], t1[1], t1[2])
	}
	return mesh
}

// TestInconsistentlyWoundMeshVolumeMagnitude proves the topological orientation path recovers the
// correct volume magnitude from a mesh whose triangles are inconsistently wound and whose shading
// normals are useless — the case the old summed-normal test got wrong (Oblikovati/Oblikovati#1318).
func TestInconsistentlyWoundMeshVolumeMagnitude(t *testing.T) {
	t.Parallel()
	const s = 2.0
	mesh := inconsistentCubeMesh(s)
	gp := meshGeometryProperties(mesh)
	want := s * s * s
	if math.Abs(gp.Volume-want) > 1e-9 {
		t.Errorf("volume = %g, want %g (topological orientation failed on inconsistent input)", gp.Volume, want)
	}
	// Centroid of a cube [0,s]³ is its centre.
	for _, c := range []struct {
		name string
		got  float64
	}{{"X", float64(gp.Centroid.X)}, {"Y", float64(gp.Centroid.Y)}, {"Z", float64(gp.Centroid.Z)}} {
		if math.Abs(c.got-s/2) > 1e-9 {
			t.Errorf("centroid.%s = %g, want %g", c.name, c.got, s/2)
		}
	}
}
