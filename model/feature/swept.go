// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/subd"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
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
			faces = append(faces, []int{s*n + i, s*n + j, ns*n + j, ns*n + i})
		}
	}
	if !closedLoop {
		faces = append(faces, reversedRow(0, n), row(k-1, n))
	}
	return subd.Mesh{Verts: verts, Faces: faces}
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
