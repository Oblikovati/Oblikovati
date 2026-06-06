// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/subd"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// sweptSolid builds a watertight solid from a sequence of cross-section loops — each
// the same number of 3D points — by connecting consecutive sections with planar quad
// side faces. When closedLoop is set the last section wraps to the first (a full
// revolution: no caps); otherwise the first and last sections are capped. The body is
// built through the sub-D cage→B-rep converter (one shared vertex/edge per cage element,
// a planar face per quad) and re-oriented outward (positive volume).
//
// This is the shared generator for revolve / sweep / loft / coil — each computes its
// section placements and hands them here. Cross-sections must not collapse onto a pole
// (a profile touching a revolve axis) in phase A; that needs pole-vertex handling.
func sweptSolid(sections [][]math.Point3, closedLoop bool, feat string) (*topo.Body, error) {
	if err := validateSections(sections, closedLoop); err != nil {
		return nil, err
	}
	mesh := sectionMesh(sections, closedLoop)
	body := subd.ToBody(mesh, feat)
	// A consistently-wound cage is either all-outward or all-inward; if the signed
	// volume came out negative the cage is inside-out, so rebuild it face-reversed.
	if ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume < 0 {
		body = subd.ToBody(reverseFaces(mesh), feat)
	}
	return body, nil
}

// validateSections rejects inputs that cannot form a solid: fewer than two sections
// (or three for a closed loop), a degenerate cross-section, or ragged section sizes.
func validateSections(sections [][]math.Point3, closedLoop bool) error {
	minSections := 2
	if closedLoop {
		minSections = 3
	}
	if len(sections) < minSections {
		return fmt.Errorf("swept solid: %d sections, need at least %d", len(sections), minSections)
	}
	n := len(sections[0])
	if n < 3 {
		return fmt.Errorf("swept solid: cross-section has %d points, need at least 3", n)
	}
	for s, sec := range sections {
		if len(sec) != n {
			return fmt.Errorf("swept solid: section %d has %d points, want %d (sections must match)", s, len(sec), n)
		}
	}
	return nil
}

// sectionMesh assembles the cage: vertex (s,i) at index s*n+i, a quad per (segment,
// cross-section edge), and start/end caps unless the sweep is a closed loop.
func sectionMesh(sections [][]math.Point3, closedLoop bool) subd.Mesh {
	k, n := len(sections), len(sections[0])
	verts := make([]math.Point3, 0, k*n)
	for _, s := range sections {
		verts = append(verts, s...)
	}
	segs := k - 1
	if closedLoop {
		segs = k
	}
	var faces [][]int
	for s := 0; s < segs; s++ {
		ns := (s + 1) % k
		for i := 0; i < n; i++ {
			j := (i + 1) % n
			a, b, c, d := s*n+i, s*n+j, ns*n+j, ns*n+i
			faces = append(faces, sideQuad(verts, a, b, c, d)...)
		}
	}
	if !closedLoop {
		faces = append(faces, reversedRow(0, n), row(k-1, n))
	}
	return subd.Mesh{Verts: verts, Faces: faces}
}

// sideQuad emits a side face between two sections as a single quad when its four corners are
// coplanar, or as two triangles when they are not. A twisted/lofted side is a warped (ruled,
// non-planar) quad; left as one face the cage→B-rep converter fits it to an APPROXIMATING plane,
// so the planar boolean's imprint segments land offset from the body's true edges and an imprint
// loop fails to close (a partial-penetration union goes non-manifold — the deformed fan blade).
// Splitting a warped quad into exact-planar triangles restores the planar-faceted invariant the
// boolean relies on. (Twisted-loft boolean defect, 2026-06.)
func sideQuad(verts []math.Point3, a, b, c, d int) [][]int {
	if quadPlanar(verts[a], verts[b], verts[c], verts[d]) {
		return [][]int{{a, b, c, d}}
	}
	return [][]int{{a, b, c}, {a, c, d}}
}

// quadPlanar reports whether the four corners lie in a common plane, within a tight absolute
// tolerance (below the arrangement's weld grid, 1e-7, so any quad the boolean would mis-imprint
// is triangulated). The deviation is the out-of-plane distance of d from the plane through a,b,c.
// tubeSolid builds a hollow swept solid (a pipe) directly from corresponding OUTER and INNER
// section rings: outer walls, inner walls (wound the opposite way so they face the bore), and —
// for an open loft — annular end caps joining the two rings. It is meshed directly rather than
// skinning the outer solid and cutting a bore, because a bore whose end caps are coplanar with
// the body's end caps leaves a Difference open (the coplanar seam doesn't stitch). The whole
// shell is built with one coherent winding, so the signed-volume flip alone picks the global
// outward orientation (the input ring winding is unknown). outerSecs and innerSecs must share
// section count and per-section point count.
func tubeSolid(outerSecs, innerSecs [][]math.Point3, closedLoop bool, feat string) (*topo.Body, error) {
	if err := validateTubeSections(outerSecs, innerSecs, closedLoop); err != nil {
		return nil, err
	}
	mesh := tubeMesh(outerSecs, innerSecs, closedLoop)
	body := subd.ToBody(mesh, feat)
	if ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume < 0 {
		body = subd.ToBody(reverseFaces(mesh), feat)
	}
	return body, nil
}

// tubeMesh connects nested outer/inner rings into a watertight tube. Inner-wall quads are the
// reverse of the outer-wall winding so the two shells stay coherently oriented (their normals
// end up pointing opposite ways); the annular caps are wound to agree with both rims (verified
// by edge-direction cancellation along each rim).
func tubeMesh(outerSecs, innerSecs [][]math.Point3, closedLoop bool) subd.Mesh {
	k, n := len(outerSecs), len(outerSecs[0])
	verts := make([]math.Point3, 0, 2*k*n)
	for _, s := range outerSecs {
		verts = append(verts, s...)
	}
	for _, s := range innerSecs {
		verts = append(verts, s...)
	}
	oi := func(s, i int) int { return s*n + i }
	ii := func(s, i int) int { return k*n + s*n + i }
	segs := k - 1
	if closedLoop {
		segs = k
	}
	var faces [][]int
	for s := 0; s < segs; s++ {
		ns := (s + 1) % k
		for i := 0; i < n; i++ {
			j := (i + 1) % n
			faces = append(faces, sideQuad(verts, oi(s, i), oi(s, j), oi(ns, j), oi(ns, i))...) // outer wall
			faces = append(faces, sideQuad(verts, ii(s, i), ii(ns, i), ii(ns, j), ii(s, j))...) // inner wall (reversed)
		}
	}
	if !closedLoop {
		for i := 0; i < n; i++ {
			j := (i + 1) % n
			faces = append(faces, sideQuad(verts, oi(0, i), ii(0, i), ii(0, j), oi(0, j))...)         // start cap
			faces = append(faces, sideQuad(verts, oi(k-1, i), oi(k-1, j), ii(k-1, j), ii(k-1, i))...) // end cap
		}
	}
	return subd.Mesh{Verts: verts, Faces: faces}
}

// validateTubeSections checks the outer/inner section sequences are skinnable and aligned: the
// outer set is a valid swept-section set, the inner set matches its section and point counts.
func validateTubeSections(outerSecs, innerSecs [][]math.Point3, closedLoop bool) error {
	if err := validateSections(outerSecs, closedLoop); err != nil {
		return err
	}
	if len(innerSecs) != len(outerSecs) {
		return fmt.Errorf("tube solid: %d inner sections, want %d (must match outer)", len(innerSecs), len(outerSecs))
	}
	n := len(outerSecs[0])
	for s, sec := range innerSecs {
		if len(sec) != n {
			return fmt.Errorf("tube solid: inner section %d has %d points, want %d (must match outer)", s, len(sec), n)
		}
	}
	return nil
}

func quadPlanar(a, b, c, d math.Point3) bool {
	nrm := a.VectorTo(b).Cross(a.VectorTo(c))
	mag := nrm.Length()
	if mag < 1e-12 {
		return true // a,b,c are collinear: no plane to deviate from (caller's quad is degenerate)
	}
	return stdmath.Abs(a.VectorTo(d).Dot(nrm)/mag) < 1e-8
}

// row returns the vertex indices of section s in order; reversedRow reverses them (the
// start cap winds opposite the side faces so the closed cage stays consistently oriented).
func row(s, n int) []int {
	out := make([]int, n)
	for i := 0; i < n; i++ {
		out[i] = s*n + i
	}
	return out
}

func reversedRow(s, n int) []int {
	out := row(s, n)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// reverseFaces returns the mesh with every face's winding reversed (flipping the whole
// solid inside-out), used to force outward orientation.
func reverseFaces(m subd.Mesh) subd.Mesh {
	faces := make([][]int, len(m.Faces))
	for fi, f := range m.Faces {
		rev := make([]int, len(f))
		for i, v := range f {
			rev[len(f)-1-i] = v
		}
		faces[fi] = rev
	}
	return subd.Mesh{Verts: m.Verts, Faces: faces}
}
