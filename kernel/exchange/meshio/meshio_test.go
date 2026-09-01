// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"errors"
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// nonManifoldSoup returns a watertight cube plus one extra triangle glued onto the cube's
// bottom-front edge (corners 0=(0,0,0), 1=(1,0,0)), so that edge is shared by THREE faces —
// the defining non-manifold signature (an edge used by more than two faces).
func nonManifoldSoup() RawMesh {
	m := cubeSoup(1)
	m.AddTriangle(math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0.5, -1, 0.5))
	return m
}

// TestSolidOrSurfaceRefusesNonManifold is the #3384 regression: a non-manifold weld is invalid
// topology that would corrupt every downstream consumer, so the import FAILS with a named decline
// (classified as ErrNonManifoldImport, naming the offending edge count) — not a usable body with a
// mere warning.
func TestSolidOrSurfaceRefusesNonManifold(t *testing.T) {
	body, warns, err := SolidOrSurface(nonManifoldSoup(), "import:test#0", DefaultWeldTolerance)
	if !errors.Is(err, ErrNonManifoldImport) {
		t.Fatalf("non-manifold import = (%v, %v, %v), want an ErrNonManifoldImport decline", body, warns, err)
	}
	if body != nil {
		t.Errorf("a refused non-manifold import returned a body; want nil so no broken body ships")
	}
	if !strings.Contains(err.Error(), "non-manifold edge") {
		t.Errorf("decline %q does not name the offending non-manifold edge count", err.Error())
	}
}

// cubeSoup returns a watertight 12-triangle triangle soup of an s-sided axis-aligned
// cube at the origin, with vertices repeated per triangle (the STL/3MF shape). Outward
// winding (CCW seen from outside) so the welded body has positive volume.
func cubeSoup(s float64) RawMesh {
	v := func(x, y, z float64) math.Point3 { return math.P3(x*s, y*s, z*s) }
	// 8 corners
	p := [8]math.Point3{
		v(0, 0, 0), v(1, 0, 0), v(1, 1, 0), v(0, 1, 0),
		v(0, 0, 1), v(1, 0, 1), v(1, 1, 1), v(0, 1, 1),
	}
	// each face as two CCW-from-outside triangles
	quads := [6][4]int{
		{0, 3, 2, 1}, // bottom (z=0), outward = -Z
		{4, 5, 6, 7}, // top (z=1), outward = +Z
		{0, 1, 5, 4}, // front (y=0), outward = -Y
		{2, 3, 7, 6}, // back (y=1), outward = +Y
		{1, 2, 6, 5}, // right (x=1), outward = +X
		{0, 4, 7, 3}, // left (x=0), outward = -X
	}
	var m RawMesh
	for _, q := range quads {
		m.AddTriangle(p[q[0]], p[q[1]], p[q[2]])
		m.AddTriangle(p[q[0]], p[q[2]], p[q[3]])
	}
	return m
}

func TestWeldCollapsesCoincidentVertices(t *testing.T) {
	raw := cubeSoup(1) // 12 triangles × 3 = 36 repeated vertices
	cage, _ := Weld(raw, DefaultWeldTolerance)
	if len(cage.Verts) != 8 {
		t.Errorf("welded vertices = %d, want 8 (a cube has 8 corners)", len(cage.Verts))
	}
	if len(cage.Faces) != 12 {
		t.Errorf("welded faces = %d, want 12", len(cage.Faces))
	}
}

func TestSolidOrSurfaceWeldsWatertightSoupIntoSolid(t *testing.T) {
	body, warns, err := SolidOrSurface(cubeSoup(2), "import:test#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("SolidOrSurface: %v", err)
	}
	if !body.IsSolid() {
		t.Fatalf("watertight cube did not import as a solid; warnings=%v", warns)
	}
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("imported cube is not a valid body: %v", r.Issues)
	}
	vol := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	if want := 8.0; stdmath.Abs(vol-want) > 1e-6 { // 2³
		t.Errorf("volume = %v, want %v", vol, want)
	}
}

func TestSolidOrSurfaceOpenSoupImportsAsSurfaceBody(t *testing.T) {
	raw := cubeSoup(1)
	raw.Tris = raw.Tris[:len(raw.Tris)-2] // drop one face's two triangles → a hole
	body, warns, err := SolidOrSurface(raw, "import:test#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("SolidOrSurface: %v", err)
	}
	if body.IsSolid() {
		t.Errorf("open mesh imported as a solid; want a surface body")
	}
	if len(warns) == 0 {
		t.Errorf("open mesh import returned no warning; want a not-watertight warning")
	}
}

func TestQualityForIsMonotonicAcrossResolutions(t *testing.T) {
	low := QualityFor(types.ResolutionLow)
	med := QualityFor(types.ResolutionMedium)
	high := QualityFor(types.ResolutionHigh)
	if !(low.ChordTolerance > med.ChordTolerance && med.ChordTolerance > high.ChordTolerance) {
		t.Errorf("chord tolerance not strictly decreasing low>med>high: %v %v %v",
			low.ChordTolerance, med.ChordTolerance, high.ChordTolerance)
	}
	if QualityFor("").ChordTolerance != med.ChordTolerance {
		t.Errorf("empty resolution did not normalize to medium")
	}
}
