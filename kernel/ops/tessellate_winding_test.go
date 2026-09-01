// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// offsetBoxBody builds a 2×2×2 box translated to corner (dx,dy,dz) by offsetting the
// cage vertices (no transform.TransformBody), isolating the tessellation winding.
func offsetBoxBody(dx, dy, dz float64) *ops.Mesh {
	m := subd.Box(2, 2, 2)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(math.V3(dx, dy, dz))
	}
	mesh, _ := ops.TessellateBody(subd.ToBody(m, "s"), ops.DefaultQuality())
	return mesh
}

// meshSignedVolume is the divergence-theorem volume of a triangle mesh; it equals the
// true volume only when every triangle is wound consistently outward.
func meshSignedVolume(m *ops.Mesh) float64 {
	v := 0.0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a := m.Positions[m.Indices[i]].AsVector()
		b := m.Positions[m.Indices[i+1]].AsVector()
		c := m.Positions[m.Indices[i+2]].AsVector()
		v += a.Dot(b.Cross(c))
	}
	return v / 6
}

// TestTessellationWindingIsOutwardAndTranslationInvariant guards the planeProjector
// orientation fix: a planar-faced solid must tessellate with consistently outward
// winding, so its divergence-theorem volume equals the analytic volume regardless of
// where it sits (a divergence sum is translation-invariant only for a watertight,
// coherently-wound mesh). Before the fix, negative-axis-normal faces were wound inward
// and an off-origin box reported 13.33 instead of 8.
func TestTessellationWindingIsOutwardAndTranslationInvariant(t *testing.T) {
	t.Parallel()
	for _, off := range [][3]float64{{0, 0, 0}, {1, 0.5, 0.5}, {10, 20, 30}, {-5, -7, -9}} {
		got := meshSignedVolume(offsetBoxBody(off[0], off[1], off[2]))
		if stdmath.Abs(got-8) > 1e-9 {
			t.Errorf("box at %v: tessellated volume = %g, want 8 (inconsistent winding?)", off, got)
		}
	}
}

func TestConcaveSingleLoopPlanarFaceVolume(t *testing.T) {
	t.Parallel()
	section := []math.Point3{
		math.P3(-0.55, 2.0, 0), math.P3(-0.8, 2.5, 0), math.P3(-2.0, 2.5, 0),
		math.P3(-2.0, 2.0, 0), math.P3(-1.025, 2.0, 0), math.P3(-0.4, 0.55, 0),
		math.P3(0.4, 0.55, 0), math.P3(1.025, 2.0, 0), math.P3(2.0, 2.0, 0),
		math.P3(2.0, 2.5, 0), math.P3(0.8, 2.5, 0), math.P3(0.55, 2.0, 0),
	}
	body := concavePrismBody(section, 3.5)
	want := polygonAreaXY(section) * 3.5
	got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	if stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("concave prism volume = %.6f, want %.6f", got, want)
	}
}

func concavePrismBody(points []math.Point3, height float64) *topo.Body {
	verts := make([]math.Point3, 0, len(points)*2)
	for _, p := range points {
		verts = append(verts, math.P3(p.X, p.Y, 0))
	}
	for _, p := range points {
		verts = append(verts, math.P3(p.X, p.Y, height))
	}

	bottom := make([]int, len(points))
	top := make([]int, len(points))
	for i := range points {
		bottom[i] = len(points) - 1 - i
		top[i] = len(points) + i
	}
	faces := [][]int{bottom, top}
	for i := range points {
		next := (i + 1) % len(points)
		faces = append(faces, []int{i, next, next + len(points), i + len(points)})
	}
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, "concave-prism")
}

func polygonAreaXY(points []math.Point3) float64 {
	var area float64
	for i := range points {
		j := (i + 1) % len(points)
		area += points[i].X * points[j].Y
		area -= points[j].X * points[i].Y
	}
	return stdmath.Abs(area) / 2
}
